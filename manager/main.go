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
	strategies := NewStrategyStore(cfg.DataDir)

	containerPool := pool.New(5)
	defer containerPool.Release()

	supaClient, err := NewSupabaseClient(cfg.SupabaseURL, cfg.SupabaseKey)
	if err != nil {
		log.Fatalf("supabase client error: %v", err)
	}

	containerMgr := NewContainerManager(cfg, credStore, containerPool)

	if err := containerMgr.EnsureNetwork(); err != nil {
		log.Fatalf("network error: %v", err)
	}

	syncer := NewSyncerWithCreds(supaClient, credStore)
	notifier := NewNotifier(cfg.DataDir, enc)
	syncer.SetNotifier(notifier)

	// Recover running containers from Docker
	allUsers := strategies.ScanUsers()
	recovered := containerMgr.RecoverUsers(allUsers)
	for _, r := range recovered {
		notifier.LoadUser(r.UserID)
	}
	log.Printf("recovered %d user containers", len(recovered))

	// Discover orphaned Docker containers not tracked in YAML
	discovered := containerMgr.DiscoverContainers()
	for uid, modes := range discovered {
		if !isValidUUID(uid) {
			continue
		}
		for _, m := range modes {
			if strategies.YAMLExists(uid, m) {
				continue
			}
			log.Printf("discovered orphaned container: %s (%s), no bbgo.yaml — stopping", uid, m)
			containerMgr.Stop(uid, m)
		}
	}


	// Periodic health check
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
			users := strategies.ScanUsers()
			for _, r := range containerMgr.CheckAndRecover(users) {
				if r.Restarted {
					notifier.Dispatch(r.UserID, NotificationEvent{
						Type:    "container",
						Title:   "Container Restarted",
						Message: fmt.Sprintf("Container bbgo-%s was restarted after an unexpected stop.", safeShortID(r.UserID)),
					})
				}
			}
			containerMgr.CleanupStopped(users)
		}
	}()

	proxy := NewBotProxy(containerMgr)

	var hub *MarketDataHub
	if h, err := NewMarketDataHub(cfg.MarketDataAddr, cfg.MarketSubscriptions); err != nil {
		log.Printf("warning: marketdata hub not available (%v), real-time data disabled", err)
	} else {
		hub = h
	}

	var testnetHub *MarketDataHub
	if cfg.MarketDataTestnetAddr != "" {
		if h, err := NewMarketDataHub(cfg.MarketDataTestnetAddr, cfg.MarketSubscriptions); err != nil {
			log.Printf("warning: testnet marketdata hub not available (%v), paper mode will use live data", err)
		} else {
			testnetHub = h
		}
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

	api := NewAPI(cfg, strategies, containerMgr, proxy, credStore, enc, syncer, hub, testnetHub, notifier, btExecutor, btJobStore)

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
	if testnetHub != nil {
		testnetHub.Close()
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
