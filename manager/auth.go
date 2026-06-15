package main

import (
	"crypto/subtle"
	"net/http"
	"regexp"
	"sync"
	"time"
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
			if r.URL.Path == "/api/health" || r.URL.Path == "/api/ws" {
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

type userRateLimiter struct {
	mu        sync.Mutex
	visitors  map[string]*visitor
	rate      time.Duration
	maxBurst  int
	lastPurge time.Time
}

type visitor struct {
	tokens   int
	lastSeen time.Time
}

// purgeInterval caps how often the full O(N) sweep runs. Without it, every
// state-changing request would scan every visitor while holding the lock.
const purgeInterval = time.Minute

func UserRateLimit(rate time.Duration, maxBurst int) func(http.Handler) http.Handler {
	rl := &userRateLimiter{
		visitors:  make(map[string]*visitor),
		rate:      rate,
		maxBurst:  maxBurst,
		lastPurge: time.Now(),
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := userIDFromRequest(r)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			if !rl.allow(userID) {
				writeError(w, http.StatusTooManyRequests, "rate limited — try again later")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (rl *userRateLimiter) allow(userID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	if now.Sub(rl.lastPurge) >= purgeInterval {
		threshold := now.Add(-1 * time.Hour)
		for id, v := range rl.visitors {
			if v.lastSeen.Before(threshold) {
				delete(rl.visitors, id)
			}
		}
		rl.lastPurge = now
	}

	v, ok := rl.visitors[userID]
	if !ok {
		rl.visitors[userID] = &visitor{tokens: rl.maxBurst - 1, lastSeen: now}
		return true
	}

	elapsed := now.Sub(v.lastSeen)
	tokensToAdd := int(elapsed / rl.rate)
	v.tokens += tokensToAdd
	if v.tokens > rl.maxBurst {
		v.tokens = rl.maxBurst
	}
	v.lastSeen = now

	if v.tokens <= 0 {
		return false
	}
	v.tokens--
	return true
}
