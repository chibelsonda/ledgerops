package httpx

import (
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/portfolio/ledgerops/backend/internal/platform/identity"
)

const maxRateLimitKeys = 10000

type rateLimitEntry struct {
	count       int
	windowStart time.Time
}

// RateLimiter provides a bounded, in-process fixed-window limiter. Deployments
// with more than one API replica should place a distributed limiter at the
// ingress as well so limits are enforced across the whole fleet.
type RateLimiter struct {
	mu         sync.Mutex
	entries    map[string]rateLimitEntry
	limit      int
	window     time.Duration
	logger     *slog.Logger
	scope      string
	now        func() time.Time
	maxEntries int
}

func NewRateLimiter(limit int, window time.Duration, scope string, logger *slog.Logger) *RateLimiter {
	return &RateLimiter{
		entries: make(map[string]rateLimitEntry), limit: limit, window: window,
		logger: logger, scope: scope, now: time.Now, maxEntries: maxRateLimitKeys,
	}
}

func (l *RateLimiter) Middleware(next http.Handler, key func(*http.Request) string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		value := key(r)
		if value == "" {
			value = "unknown"
		}
		allowed, remaining, retryAfter := l.allow(value)
		w.Header().Set("RateLimit-Limit", strconv.Itoa(l.limit))
		w.Header().Set("RateLimit-Remaining", strconv.Itoa(remaining))
		if !allowed {
			seconds := max(int((retryAfter+time.Second-1)/time.Second), 1)
			w.Header().Set("Retry-After", strconv.Itoa(seconds))
			l.logger.Warn("security rate limit exceeded", "scope", l.scope, "client", value, "path", r.URL.Path)
			WriteError(w, http.StatusTooManyRequests, "rate_limit_exceeded", "Too many requests. Wait before trying again.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (l *RateLimiter) allow(key string) (bool, int, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	entry, exists := l.entries[key]
	if exists && now.Sub(entry.windowStart) >= l.window {
		delete(l.entries, key)
		exists = false
	}
	if !exists {
		if len(l.entries) >= l.maxEntries {
			l.prune(now)
		}
		if len(l.entries) >= l.maxEntries {
			return false, 0, l.window
		}
		entry = rateLimitEntry{windowStart: now}
	}
	if entry.count >= l.limit {
		return false, 0, l.window - now.Sub(entry.windowStart)
	}
	entry.count++
	l.entries[key] = entry
	return true, l.limit - entry.count, l.window - now.Sub(entry.windowStart)
}

func (l *RateLimiter) prune(now time.Time) {
	for key, entry := range l.entries {
		if now.Sub(entry.windowStart) >= l.window {
			delete(l.entries, key)
		}
	}
}

type ClientIPResolver struct {
	trusted []netip.Prefix
}

func NewClientIPResolver(cidrs []string) (*ClientIPResolver, error) {
	resolver := &ClientIPResolver{trusted: make([]netip.Prefix, 0, len(cidrs))}
	for _, cidr := range cidrs {
		prefix, err := netip.ParsePrefix(cidr)
		if err != nil {
			return nil, err
		}
		resolver.trusted = append(resolver.trusted, prefix)
	}
	return resolver, nil
}

func (r *ClientIPResolver) Key(request *http.Request) string {
	peer := parseRemoteIP(request.RemoteAddr)
	if !peer.IsValid() || !r.isTrusted(peer) {
		return peer.String()
	}
	forwarded := strings.Split(request.Header.Get("X-Forwarded-For"), ",")
	current := peer
	for index := len(forwarded) - 1; index >= 0 && r.isTrusted(current); index-- {
		candidate, err := netip.ParseAddr(strings.TrimSpace(forwarded[index]))
		if err != nil {
			return peer.String()
		}
		current = candidate.Unmap()
	}
	return current.String()
}

func (r *ClientIPResolver) isTrusted(address netip.Addr) bool {
	for _, prefix := range r.trusted {
		if prefix.Contains(address) {
			return true
		}
	}
	return false
}

func parseRemoteIP(remoteAddress string) netip.Addr {
	host, _, err := net.SplitHostPort(remoteAddress)
	if err != nil {
		host = remoteAddress
	}
	address, _ := netip.ParseAddr(strings.TrimSpace(host))
	return address.Unmap()
}

func PrincipalRateLimitKey(r *http.Request) string {
	principal, ok := identity.FromContext(r.Context())
	if !ok {
		return "unauthenticated"
	}
	return principal.UserID
}
