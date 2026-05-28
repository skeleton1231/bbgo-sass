package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/c9s/bbgo/saas/manager/pool"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	enc, err := NewEncryptor(cfg.EncryptionKey)
	if err != nil {
		log.Fatalf("encryption error: %v", err)
	}

	credStore := NewCredentialStore(cfg.DataDir, enc)
	users := NewUserContainerManager()

	containerPool := pool.New(5)
	defer containerPool.Release()
	syncPool := pool.New(5)
	defer syncPool.Release()

	containerMgr := NewContainerManager(cfg, credStore, containerPool)

	if err := containerMgr.EnsureNetwork(); err != nil {
		log.Fatalf("network error: %v", err)
	}

	if err := RunMigration(cfg.SupabaseDBURL); err != nil {
		log.Printf("warning: auto-migration failed: %v", err)
	}

	syncer := NewSyncerWithCreds(users, cfg, containerMgr, credStore, syncPool)

	notifier := NewNotifier(cfg.DataDir, enc)

	recoveredUsers, err := syncer.LoadUsersFromSupabase()
	if err != nil {
		log.Printf("warning: could not load users from supabase: %v", err)
	} else {
		users.Restore(recoveredUsers)
		recoveryResults := containerMgr.RecoverUsers(recoveredUsers)
		for _, r := range recoveryResults {
			users.UpdateStatus(r.UserID, r.Mode, r.Status)
		}
		for _, uc := range recoveredUsers {
			notifier.LoadUser(uc.UserID)
		}
		log.Printf("restored %d users from supabase", len(recoveredUsers))
	}

	// Discover orphaned Docker containers not tracked in Supabase
	discovered := containerMgr.DiscoverContainers()
	for uid, modes := range discovered {
		for _, m := range modes {
			if _, exists := users.Get(uid, m); exists {
				continue
			}
			log.Printf("discovered orphaned container: %s (%s), registering", uid, m)
			users.AddStrategy(uid, m, StrategyEntry{})
			users.UpdateStatus(uid, m, StatusRunning)
		}
	}

	syncer.SyncAll()

	// Auto-sync backtest data on startup (background, non-blocking)
	btSyncPool := pool.New(2)
	defer btSyncPool.Release()
	go func() {
		time.Sleep(30 * time.Second)
		for _, ex := range cfg.BacktestExchanges {
			for _, sym := range cfg.BacktestSymbols {
				ex, sym := ex, sym
				if err := btSyncPool.Submit(func() {
					log.Printf("auto-syncing backtest data: %s/%s", ex, sym)
					if out, err := containerMgr.SyncBacktest(ex, sym, cfg.BacktestStartTime, cfg.BacktestEndTime); err != nil {
						log.Printf("backtest auto-sync %s/%s failed: %v (output: %s)", ex, sym, err, out)
					} else {
						log.Printf("backtest auto-sync %s/%s done", ex, sym)
					}
				}); err != nil {
					log.Printf("backtest auto-sync submit %s/%s: %v", ex, sym, err)
				}
			}
		}
		btSyncPool.Wait()
	}()

	// Periodic sync and health check
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
			}
			syncer.SyncAll()
			allUsers := users.ListUsers()
			for _, r := range containerMgr.CheckAndRecover(allUsers) {
				if r.Error != "" {
					users.UpdateStatus(r.UserID, r.Mode, StatusError)
				}
				if r.Restarted {
					notifier.Dispatch(r.UserID, NotificationEvent{
						Type:    "container",
						Title:   "Container Restarted",
						Message: fmt.Sprintf("Container bbgo-%s was restarted after an unexpected stop.", safeShortID(r.UserID)),
					})
				}
			}
		}
	}()

	proxy := NewBotProxy(containerMgr)

	var hub *MarketDataHub
	if h, err := NewMarketDataHub(cfg.MarketDataAddr, cfg.MarketSubscriptions); err != nil {
		log.Printf("warning: marketdata hub not available (%v), real-time data disabled", err)
	} else {
		hub = h
	}

	btJobStore := NewBacktestJobStore(cfg.DataDir)
	btExecutor := NewBacktestExecutor(btJobStore, containerMgr, notifier)

	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				btJobStore.Prune(24 * time.Hour)
			}
		}
	}()

	api := NewAPI(cfg, users, containerMgr, proxy, credStore, enc, syncer, hub, notifier, btExecutor, btJobStore)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(maxBodySize(2 << 20)) // 2MB max request body

	r.Use(SharedSecretAuth(cfg.ManagerToken))

	api.RegisterRoutes(r)

	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := &http.Server{Addr: addr, Handler: r}

	go func() {
		log.Printf("Manager starting on %s (docker network: %s, image: %s)", addr, cfg.DockerNetwork, cfg.BBGOImage)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")
	close(done)
	api.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
	if hub != nil {
		hub.Close()
	}
	log.Println("server stopped")
}

func maxBodySize(n int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, n)
			next.ServeHTTP(w, r)
		})
	}
}
