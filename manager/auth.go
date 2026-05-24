package main

import (
	"crypto/subtle"
	"net/http"
	"regexp"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func isValidUUID(s string) bool {
	return uuidPattern.MatchString(s)
}

func safeShortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

func userIDFromRequest(r *http.Request) (string, bool) {
	id := r.Header.Get("X-User-Id")
	if id == "" {
		return "", false
	}
	if !isValidUUID(id) {
		return "", false
	}
	return id, true
}

func SharedSecretAuth(sharedSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/health" {
				next.ServeHTTP(w, r)
				return
			}
			token := r.Header.Get("X-Manager-Token")
			if token == "" {
				token = r.URL.Query().Get("token")
			}
			if subtle.ConstantTimeCompare([]byte(token), []byte(sharedSecret)) != 1 {
				writeError(w, http.StatusUnauthorized, "invalid or missing manager token")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
