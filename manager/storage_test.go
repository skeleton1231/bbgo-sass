package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStorageClient_Upload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "backtest-reports") {
			t.Errorf("path should contain bucket name: %s", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-key" {
			t.Errorf("expected Bearer auth, got %s", auth)
		}
		if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "multipart/form-data") {
			t.Errorf("expected multipart content type, got %s", ct)
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()

	sc := NewStorageClient(srv.URL, "test-key")
	if err := sc.Upload("user1", "job1", "trades.tsv", []byte("data")); err != nil {
		t.Fatalf("Upload: %v", err)
	}
}

func TestStorageClient_Upload_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		fmt.Fprint(w, "internal error")
	}))
	defer srv.Close()

	sc := NewStorageClient(srv.URL, "test-key")
	err := sc.Upload("user1", "job1", "trades.tsv", []byte("data"))
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should mention status 500: %v", err)
	}
}

func TestStorageClient_CreateSignedURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var req map[string]int
		json.NewDecoder(r.Body).Decode(&req)
		if req["expiresIn"] != 3600 {
			t.Errorf("expected expiresIn=3600, got %d", req["expiresIn"])
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(signedURLResponse{SignedURL: "/storage/v1/object/sign/backtest-reports/user1/job1/file.tsv?token=abc"})
	}))
	defer srv.Close()

	sc := NewStorageClient(srv.URL, "test-key")
	url, err := sc.CreateSignedURL("user1", "job1", "file.tsv", 3600)
	if err != nil {
		t.Fatalf("CreateSignedURL: %v", err)
	}
	if !strings.Contains(url, srv.URL) {
		t.Errorf("signed URL should contain base URL: %s", url)
	}
	if !strings.Contains(url, "token=abc") {
		t.Errorf("signed URL should contain token: %s", url)
	}
}

func TestStorageClient_CreateSignedURL_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		fmt.Fprint(w, "forbidden")
	}))
	defer srv.Close()

	sc := NewStorageClient(srv.URL, "test-key")
	_, err := sc.CreateSignedURL("user1", "job1", "f.tsv", 3600)
	if err == nil {
		t.Fatal("expected error for 403")
	}
}

func TestStorageClient_RemoveFolder(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()

	sc := NewStorageClient(srv.URL, "test-key")
	sc.RemoveFolder("user1", "job1")
	if !called {
		t.Error("expected DELETE request")
	}
}

func TestStorageClient_RemoveFolder_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		fmt.Fprint(w, "fail")
	}))
	defer srv.Close()

	sc := NewStorageClient(srv.URL, "test-key")
	sc.RemoveFolder("user1", "job1") // should not panic
}
