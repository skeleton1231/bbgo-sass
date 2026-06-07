package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- verifyCredential unit tests with mocked exchange servers ---

func TestVerifyCredential_BinanceLive_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v3/account" && r.Header.Get("X-MBX-APIKEY") != "" && r.URL.Query().Get("signature") != "" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	origURLs := exchangeBaseURLs
	exchangeBaseURLs = map[string]struct{ live, testnet string }{
		"binance": {live: srv.URL, testnet: srv.URL},
	}
	defer func() { exchangeBaseURLs = origURLs }()

	result := verifyCredential("binance", "test-key", "test-secret", "", false)
	if !result.Verified {
		t.Errorf("expected verified=true, got error: %s", result.Error)
	}
}

func TestVerifyCredential_BinanceTestnet_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v3/account" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	origURLs := exchangeBaseURLs
	exchangeBaseURLs = map[string]struct{ live, testnet string }{
		"binance": {live: "http://localhost:1", testnet: srv.URL},
	}
	defer func() { exchangeBaseURLs = origURLs }()

	result := verifyCredential("binance", "key", "secret", "", true)
	if !result.Verified {
		t.Errorf("expected verified=true for testnet, got error: %s", result.Error)
	}
}

func TestVerifyCredential_Binance_InvalidKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"code":-2015,"msg":"Invalid API-key."}`))
	}))
	defer srv.Close()

	origURLs := exchangeBaseURLs
	exchangeBaseURLs = map[string]struct{ live, testnet string }{
		"binance": {live: srv.URL, testnet: srv.URL},
	}
	defer func() { exchangeBaseURLs = origURLs }()

	result := verifyCredential("binance", "bad-key", "bad-secret", "", false)
	if result.Verified {
		t.Error("expected verified=false for invalid key")
	}
	if !strings.Contains(result.Error, "401") {
		t.Errorf("expected 401 in error, got: %s", result.Error)
	}
}

func TestVerifyCredential_UnsupportedExchange(t *testing.T) {
	result := verifyCredential("unknown_exchange", "k", "s", "", false)
	if result.Verified {
		t.Error("expected verified=false for unsupported exchange")
	}
	if !strings.Contains(result.Error, "unsupported exchange") {
		t.Errorf("expected 'unsupported exchange' error, got: %s", result.Error)
	}
}

func TestVerifyCredential_ConnectionFailed(t *testing.T) {
	origURLs := exchangeBaseURLs
	exchangeBaseURLs = map[string]struct{ live, testnet string }{
		"binance": {live: "http://localhost:1", testnet: "http://localhost:1"},
	}
	defer func() { exchangeBaseURLs = origURLs }()

	result := verifyCredential("binance", "k", "s", "", false)
	if result.Verified {
		t.Error("expected verified=false when connection fails")
	}
	if !strings.Contains(result.Error, "connection failed") {
		t.Errorf("expected connection error, got: %s", result.Error)
	}
}

func TestVerifyCredential_Bybit_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-BAPI-API-KEY") != "" && r.Header.Get("X-BAPI-SIGN") != "" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	origURLs := exchangeBaseURLs
	exchangeBaseURLs = map[string]struct{ live, testnet string }{
		"bybit": {live: srv.URL, testnet: srv.URL},
	}
	defer func() { exchangeBaseURLs = origURLs }()

	result := verifyCredential("bybit", "key", "secret", "", false)
	if !result.Verified {
		t.Errorf("expected verified=true for bybit, got error: %s", result.Error)
	}
}

func TestVerifyCredential_Bitget_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("ACCESS-KEY") != "" && r.Header.Get("ACCESS-SIGN") != "" && r.Header.Get("ACCESS-PASSPHRASE") != "" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	origURLs := exchangeBaseURLs
	exchangeBaseURLs = map[string]struct{ live, testnet string }{
		"bitget": {live: srv.URL, testnet: srv.URL},
	}
	defer func() { exchangeBaseURLs = origURLs }()

	result := verifyCredential("bitget", "key", "secret", "mypass", false)
	if !result.Verified {
		t.Errorf("expected verified=true for bitget, got error: %s", result.Error)
	}
}

func TestVerifyCredential_Bitget_NoPassphrase(t *testing.T) {
	result := verifyCredential("bitget", "key", "secret", "", false)
	if result.Verified {
		t.Error("expected verified=false for bitget without passphrase")
	}
	if !strings.Contains(result.Error, "passphrase") {
		t.Errorf("expected passphrase error, got: %s", result.Error)
	}
}

// --- CreateCredential integration tests with verification ---

func TestCreateCredential_Verified_SetsIsVerified(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	origURLs := exchangeBaseURLs
	exchangeBaseURLs = map[string]struct{ live, testnet string }{
		"binance": {live: srv.URL, testnet: srv.URL},
	}
	defer func() { exchangeBaseURLs = origURLs }()

	api, cleanup := setupCredsAPI(t)
	api.verifyCredFn = nil
	defer cleanup()

	body := `{"exchange":"binance","api_key":"valid-key","api_secret":"valid-secret"}`
	req := httptest.NewRequest(http.MethodPost, "/api/credentials", strings.NewReader(body))
	req.Header.Set("X-User-Id", credsUID)
	w := httptest.NewRecorder()
	api.CreateCredential(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["is_verified"] != true {
		t.Errorf("expected is_verified=true, got %v", resp["is_verified"])
	}
	if _, hasErr := resp["verify_error"]; hasErr {
		t.Error("expected no verify_error for successful verification")
	}
}

func TestCreateCredential_VerificationFailed_SetsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"code":-2015,"msg":"Invalid API-key."}`))
	}))
	defer srv.Close()

	origURLs := exchangeBaseURLs
	exchangeBaseURLs = map[string]struct{ live, testnet string }{
		"binance": {live: srv.URL, testnet: srv.URL},
	}
	defer func() { exchangeBaseURLs = origURLs }()

	api, cleanup := setupCredsAPI(t)
	api.verifyCredFn = nil
	defer cleanup()

	body := `{"exchange":"binance","api_key":"bad-key","api_secret":"bad-secret"}`
	req := httptest.NewRequest(http.MethodPost, "/api/credentials", strings.NewReader(body))
	req.Header.Set("X-User-Id", credsUID)
	w := httptest.NewRecorder()
	api.CreateCredential(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 (created but unverified), got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["is_verified"] != false {
		t.Errorf("expected is_verified=false, got %v", resp["is_verified"])
	}
	if errStr, ok := resp["verify_error"].(string); !ok || !strings.Contains(errStr, "401") {
		t.Errorf("expected verify_error with 401, got %v", resp["verify_error"])
	}
}

func TestCreateCredential_Unverified_DoesNotRestart(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	origURLs := exchangeBaseURLs
	exchangeBaseURLs = map[string]struct{ live, testnet string }{
		"binance": {live: srv.URL, testnet: srv.URL},
	}
	defer func() { exchangeBaseURLs = origURLs }()

	api, cleanup := setupCredsAPI(t)
	api.verifyCredFn = nil
	defer cleanup()

	createTestInstance(t, api.store, credsUID, ModeLive, "grid2", "BTCUSDT", nil)
	api.container.checkRunningFn = func(string) (bool, error) { return true, nil }

	var dockerCalled bool
	api.container.dockerFn = func(args ...string) (string, error) {
		dockerCalled = true
		return "", nil
	}

	body := `{"exchange":"binance","api_key":"k","api_secret":"s"}`
	req := httptest.NewRequest(http.MethodPost, "/api/credentials", strings.NewReader(body))
	req.Header.Set("X-User-Id", credsUID)
	w := httptest.NewRecorder()
	api.CreateCredential(w, req)

	if dockerCalled {
		t.Error("unverified credential should NOT trigger container restart")
	}
}

// --- StartUserContainer checks IsVerified ---

func TestStartUserContainer_UnverifiedCredential_Rejected(t *testing.T) {
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(dir, enc)

	keyEnc, _ := enc.Encrypt("test-key")
	secretEnc, _ := enc.Encrypt("test-secret")
	creds.Upsert(ExchangeCredential{
		UserID:             testUUID,
		Exchange:           "binance",
		APIKeyEncrypted:    keyEnc,
		APISecretEncrypted: secretEnc,
		IsTestnet:          false,
		IsVerified:         false,
	})

	store := NewInstanceStore(dir, testRegistry)
	createTestInstance(t, store, testUUID, ModeLive, "grid2", "BTCUSDT", nil)

	cfg := &Config{ManagerToken: "test-token", DataDir: dir}
	cm := NewContainerManager(cfg, nil, nil, store)
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, store, cm, proxy, creds, enc, nil, nil, nil, nil, nil, nil, nil)

	r := testRouter(api)
	req := httptest.NewRequest("POST", "/api/users/"+testUUID+"/start?mode=live", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unverified credential, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "not verified") {
		t.Errorf("expected 'not verified' error, got: %s", w.Body.String())
	}
}

func TestStartUserContainer_VerifiedCredential_Allowed(t *testing.T) {
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(dir, enc)

	keyEnc, _ := enc.Encrypt("test-key")
	secretEnc, _ := enc.Encrypt("test-secret")
	creds.Upsert(ExchangeCredential{
		UserID:             testUUID,
		Exchange:           "binance",
		APIKeyEncrypted:    keyEnc,
		APISecretEncrypted: secretEnc,
		IsTestnet:          false,
		IsVerified:         true,
	})

	store := NewInstanceStore(dir, testRegistry)
	createTestInstance(t, store, testUUID, ModeLive, "grid2", "BTCUSDT", nil)

	cfg := &Config{ManagerToken: "test-token", DataDir: dir}
	cm := NewContainerManager(cfg, nil, nil, store)
	cm.checkRunningFn = func(string) (bool, error) { return true, nil }
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, store, cm, proxy, creds, enc, nil, nil, nil, nil, nil, nil, nil)

	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"message": "pong"})
	}))
	defer bbgoSrv.Close()
	api.newBBGoClient = func(_ string) *BBGoClient {
		return NewBBGoClient(bbgoSrv.URL)
	}

	r := testRouter(api)
	req := httptest.NewRequest("POST", "/api/users/"+testUUID+"/start?mode=live", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted && w.Code != http.StatusOK {
		t.Fatalf("expected 200/202 for verified credential, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStartUserContainer_NoCredsStore_NoVerificationCheck(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, testRegistry)
	createTestInstance(t, store, testUUID, ModeLive, "grid2", "BTCUSDT", nil)

	cfg := &Config{ManagerToken: "test-token", DataDir: dir}
	cm := NewContainerManager(cfg, nil, nil, store)
	cm.checkRunningFn = func(string) (bool, error) { return true, nil }
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, store, cm, proxy, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"message": "pong"})
	}))
	defer bbgoSrv.Close()
	api.newBBGoClient = func(_ string) *BBGoClient {
		return NewBBGoClient(bbgoSrv.URL)
	}

	r := testRouter(api)
	req := httptest.NewRequest("POST", "/api/users/"+testUUID+"/start?mode=live", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted && w.Code != http.StatusOK {
		t.Fatalf("expected 200/202 when no cred store, got %d: %s", w.Code, w.Body.String())
	}
}

// --- GetByMode unit tests ---

func TestCredentialStore_GetByMode(t *testing.T) {
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	cs := NewCredentialStore(dir, enc)

	keyEnc, _ := enc.Encrypt("live-key")
	secretEnc, _ := enc.Encrypt("live-secret")
	cs.Upsert(ExchangeCredential{
		UserID:             "user1",
		Exchange:           "binance",
		APIKeyEncrypted:    keyEnc,
		APISecretEncrypted: secretEnc,
		IsTestnet:          false,
		IsVerified:         true,
	})

	keyEnc2, _ := enc.Encrypt("test-key")
	secretEnc2, _ := enc.Encrypt("test-secret")
	cs.Upsert(ExchangeCredential{
		UserID:             "user1",
		Exchange:           "binance",
		APIKeyEncrypted:    keyEnc2,
		APISecretEncrypted: secretEnc2,
		IsTestnet:          true,
		IsVerified:         false,
	})

	live, err := cs.GetByMode("user1", "binance", false)
	if err != nil {
		t.Fatalf("GetByMode live: %v", err)
	}
	if !live.IsVerified {
		t.Error("expected live credential to be verified")
	}
	if live.IsTestnet {
		t.Error("expected live credential to have IsTestnet=false")
	}

	testnet, err := cs.GetByMode("user1", "binance", true)
	if err != nil {
		t.Fatalf("GetByMode testnet: %v", err)
	}
	if testnet.IsVerified {
		t.Error("expected testnet credential to be unverified")
	}
	if !testnet.IsTestnet {
		t.Error("expected testnet credential to have IsTestnet=true")
	}

	_, err = cs.GetByMode("user1", "okex", false)
	if err == nil {
		t.Error("expected error for non-existent exchange")
	}
}

func TestExchangeHasVerifier(t *testing.T) {
	for _, tc := range []struct {
		exchange string
		want     bool
	}{
		{"binance", true},
		{"bybit", true},
		{"bitget", true},
		{"okex", false},
		{"kucoin", false},
		{"max", false},
		{"coinbase", false},
		{"bitfinex", false},
	} {
		if got := exchangeHasVerifier(tc.exchange); got != tc.want {
			t.Errorf("exchangeHasVerifier(%q) = %v, want %v", tc.exchange, got, tc.want)
		}
	}
}

func TestVerifyCredential_NotImplementedExchange(t *testing.T) {
	result := verifyCredential("okex", "key", "secret", "", false)
	if result.Verified {
		t.Error("expected verified=false for unimplemented exchange")
	}
	if !strings.Contains(result.Error, "not implemented") {
		t.Errorf("expected 'not implemented' error, got: %s", result.Error)
	}
}

func TestVerifyCredential_NotImplementedTestnet(t *testing.T) {
	result := verifyCredential("kucoin", "key", "secret", "", true)
	if result.Verified {
		t.Error("expected verified=false for unimplemented exchange testnet")
	}
	if !strings.Contains(result.Error, "not implemented") {
		t.Errorf("expected 'not implemented' error, got: %s", result.Error)
	}
}

func TestStartUserContainer_UnsupportedExchange_SkipsVerification(t *testing.T) {
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(dir, enc)

	keyEnc, _ := enc.Encrypt("test-key")
	secretEnc, _ := enc.Encrypt("test-secret")
	creds.Upsert(ExchangeCredential{
		UserID:             testUUID,
		Exchange:           "okex",
		APIKeyEncrypted:    keyEnc,
		APISecretEncrypted: secretEnc,
		IsTestnet:          false,
		IsVerified:         false,
	})

	store := NewInstanceStore(dir, testRegistry)
	inst := &StrategyInstance{
		UserID:   testUUID,
		Mode:     ModeLive,
		Strategy: "grid2",
		Exchange: "okex",
		Symbol:   "BTCUSDT",
		Config:   rawJSON("{}"),
	}
	inst.InstanceID = computeInstanceID(inst.Strategy, inst.Symbol, inst.Config)
	store.CreateInstance(inst, func(string) bool { return false })

	cfg := &Config{ManagerToken: "test-token", DataDir: dir}
	cm := NewContainerManager(cfg, nil, nil, store)
	cm.checkRunningFn = func(string) (bool, error) { return true, nil }
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, store, cm, proxy, creds, enc, nil, nil, nil, nil, nil, nil, nil)

	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"message": "pong"})
	}))
	defer bbgoSrv.Close()
	api.newBBGoClient = func(_ string) *BBGoClient {
		return NewBBGoClient(bbgoSrv.URL)
	}

	r := testRouter(api)
	req := httptest.NewRequest("POST", "/api/users/"+testUUID+"/start?mode=live", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted && w.Code != http.StatusOK {
		t.Fatalf("expected 200/202 for unverified OKX key (no verifier), got %d: %s", w.Code, w.Body.String())
	}
}
