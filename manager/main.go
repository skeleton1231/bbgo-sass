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
	containerMgr := NewContainerManager(cfg, credStore)

	if err := containerMgr.EnsureNetwork(); err != nil {
		log.Fatalf("network error: %v", err)
	}

	if err := RunMigration(cfg.SupabaseDBURL); err != nil {
		log.Printf("warning: auto-migration failed: %v", err)
	}

	syncer := NewSyncerWithCreds(users, cfg, containerMgr, credStore)

	recoveredUsers, err := syncer.LoadUsersFromSupabase()
	if err != nil {
		log.Printf("warning: could not load users from supabase: %v", err)
	} else {
		users.Restore(recoveredUsers)
		containerMgr.RecoverUsers(recoveredUsers)
		log.Printf("restored %d users from supabase", len(recoveredUsers))
	}

	syncer.SyncAll()

	// Periodic sync and health check
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			syncer.SyncAll()
			users := users.ListUsers()
			for _, uc := range users {
				if uc.Status == StatusRunning && !containerMgr.IsRunning(uc.UserID) {
					log.Printf("health check: container %s died, restarting", containerMgr.containerName(uc.UserID))
					if err := containerMgr.CreateAndStart(uc); err != nil {
						log.Printf("health check: restart %s failed: %v", uc.UserID, err)
					}
				}
			}
		}
	}()

	proxy := NewBotProxy(containerMgr)
	api := NewAPI(users, containerMgr, proxy, credStore, enc, syncer)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

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
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
	log.Println("server stopped")
}
