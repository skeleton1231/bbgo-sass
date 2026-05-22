package main

import (
	"fmt"
	"log"
	"net/http"
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

	syncer := NewSyncer(users, cfg, containerMgr)

	recoveredUsers, err := syncer.LoadUsersFromSupabase()
	if err != nil {
		log.Printf("warning: could not load users from supabase: %v", err)
	} else {
		users.Restore(recoveredUsers)
		containerMgr.RecoverUsers(recoveredUsers)
		log.Printf("restored %d users from supabase", len(recoveredUsers))
	}

	syncer.SyncAll()

	proxy := NewBotProxy(containerMgr)
	api := NewAPI(users, containerMgr, proxy, credStore, enc, syncer)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	api.RegisterRoutes(r)

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("Manager starting on %s (docker network: %s, image: %s)", addr, cfg.DockerNetwork, cfg.BBGOImage)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
