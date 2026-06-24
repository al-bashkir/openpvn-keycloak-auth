package httpserver

import (
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/al-bashkir/openvpn-keycloak-auth/internal/logsanitize"
)

// loggingMiddleware logs HTTP requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		slog.Info("http request", // #nosec G706 -- values sanitized via logsanitize.Sanitize
			"method", logsanitize.Sanitize(r.Method),
			"path", redactRequestPath(r.URL.Path),
			"remote_addr", logsanitize.Sanitize(r.RemoteAddr),
			"user_agent", logsanitize.Sanitize(r.Header.Get("User-Agent")),
		)

		next.ServeHTTP(w, r)

		slog.Debug("http request completed", // #nosec G706 -- values sanitized via logsanitize.Sanitize
			"method", logsanitize.Sanitize(r.Method),
			"path", redactRequestPath(r.URL.Path),
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

// recoveryMiddleware recovers from panics
func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic recovered",
					"error", err,
					"stack", string(debug.Stack()),
				)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// ipEntry stores a rate limiter and the last time it was accessed.
type ipEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// IPRateLimiter implements per-IP rate limiting with TTL-based eviction.
type IPRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*ipEntry
	rate     rate.Limit
	burst    int
	ttl      time.Duration // entries are evicted after this duration of inactivity
	maxSize  int           // maximum number of tracked IPs
}

func newIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	rl := &IPRateLimiter{
		limiters: make(map[string]*ipEntry),
		rate:     r,
		burst:    b,
		ttl:      5 * time.Minute,
		maxSize:  10000,
	}

	// Start background eviction goroutine
	go rl.evictLoop()

	return rl
}

func (i *IPRateLimiter) getLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	entry, exists := i.limiters[ip]
	if exists {
		entry.lastSeen = time.Now()
		return entry.limiter
	}

	// Evict oldest entries if at capacity
	if len(i.limiters) >= i.maxSize {
		i.evictOldest()
	}

	limiter := rate.NewLimiter(i.rate, i.burst)
	i.limiters[ip] = &ipEntry{
		limiter:  limiter,
		lastSeen: time.Now(),
	}

	return limiter
}

// evictLoop periodically removes stale entries.
func (i *IPRateLimiter) evictLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		i.mu.Lock()
		now := time.Now()
		for ip, entry := range i.limiters {
			if now.Sub(entry.lastSeen) > i.ttl {
				delete(i.limiters, ip)
			}
		}
		i.mu.Unlock()
	}
}

// evictOldest removes the oldest entry. Must be called with mu held.
func (i *IPRateLimiter) evictOldest() {
	var oldestIP string
	var oldestTime time.Time

	for ip, entry := range i.limiters {
		if oldestIP == "" || entry.lastSeen.Before(oldestTime) {
			oldestIP = ip
			oldestTime = entry.lastSeen
		}
	}

	if oldestIP != "" {
		delete(i.limiters, oldestIP)
	}
}

// Global rate limiter: 10 requests per second per IP, burst of 50.
// ponytail: process-lifetime singleton; its evictLoop goroutine intentionally
// has no stop channel. Add one only if rate limiting becomes per-Server.
var globalLimiter = newIPRateLimiter(10, 50)

// rateLimitMiddleware implements rate limiting
func rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := extractIP(r)
		limiter := globalLimiter.getLimiter(ip)

		if !limiter.Allow() {
			slog.Warn("rate limit exceeded", // #nosec G706 -- values sanitized via logsanitize.Sanitize
				"ip", logsanitize.Sanitize(ip),
				"path", redactRequestPath(r.URL.Path),
			)
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// extractIP extracts the client IP from the request.
// Only uses RemoteAddr by default to prevent spoofing via X-Forwarded-For.
// If this service is behind a trusted reverse proxy, configure the proxy
// to set X-Real-IP and update this function accordingly.
func extractIP(r *http.Request) string {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func redactRequestPath(path string) string {
	idx := strings.LastIndex(path, "/auth/")
	if idx != -1 {
		return logsanitize.Sanitize(path[:idx] + "/auth/{state}")
	}
	return logsanitize.Sanitize(path)
}

// securityHeadersMiddleware adds security headers to responses.
//
// X-XSS-Protection is intentionally not set: modern browsers (Chrome/Edge)
// removed it and OWASP recommends omitting it because "1; mode=block" can
// introduce side-channel XSS in some legacy clients. CSP is the active
// mitigation.
func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'")
		if r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		next.ServeHTTP(w, r)
	})
}
