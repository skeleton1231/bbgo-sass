package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBBGoClient_LimitsResponseSize(t *testing.T) {
	bigBody := strings.Repeat(`{"data":"`+strings.Repeat("x", 1024)+`"}`, 5000)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(bigBody))
	}))
	defer srv.Close()

	client := NewBBGoClient(srv.URL)
	// Ping tries to decode JSON — huge body should be truncated without OOM
	err := client.Ping()
	_ = err
}

func TestBBGoClient_ClientTimeoutOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Millisecond)
		json.NewEncoder(w).Encode(map[string]string{"message": "pong"})
	}))
	defer srv.Close()

	client := NewBBGoClient(srv.URL)
	if err := client.Ping(); err != nil {
		t.Fatalf("Ping() should succeed: %v", err)
	}
}

func TestBBGoClient_ErrorBodyLimit(t *testing.T) {
	bigError := strings.Repeat("error detail ", 10000)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(bigError))
	}))
	defer srv.Close()

	client := NewBBGoClient(srv.URL)
	err := client.Ping()
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	errStr := err.Error()
	if len(errStr) > 8192 {
		t.Errorf("error message too long (%d bytes), should be truncated", len(errStr))
	}
}
