package interceptors

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"connectrpc.com/connect"
)

type rateWindow struct {
	started time.Time
	count   int
}

// RateLimiter applies fixed one-minute windows per procedure and client IP.
type RateLimiter struct {
	mu          sync.Mutex
	windows     map[string]rateWindow
	limits      map[string]int
	lastCleanup time.Time
}

func NewRateLimiter(limits map[string]int) *RateLimiter {
	return &RateLimiter{
		windows: make(map[string]rateWindow),
		limits:  limits,
	}
}

func (limiter *RateLimiter) UnaryInterceptor() connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, request connect.AnyRequest) (connect.AnyResponse, error) {
			procedure := request.Spec().Procedure
			limit, applies := limiter.limits[procedure]
			if applies && !limiter.allow(clientIP(request.Peer().Addr), procedure, limit, time.Now()) {
				return nil, connect.NewError(connect.CodeResourceExhausted, errors.New("rate limit exceeded"))
			}
			return next(ctx, request)
		}
	})
}

func (limiter *RateLimiter) allow(ip, procedure string, limit int, now time.Time) bool {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	if now.Sub(limiter.lastCleanup) >= time.Minute {
		for key, window := range limiter.windows {
			if now.Sub(window.started) >= time.Minute {
				delete(limiter.windows, key)
			}
		}
		limiter.lastCleanup = now
	}

	key := procedure + ":" + ip
	window, exists := limiter.windows[key]
	if !exists || now.Sub(window.started) >= time.Minute {
		limiter.windows[key] = rateWindow{started: now, count: 1}
		return true
	}
	if window.count >= limit {
		return false
	}
	window.count++
	limiter.windows[key] = window
	return true
}

func clientIP(address string) string {
	if address == "" {
		return "unknown"
	}
	if host, _, err := net.SplitHostPort(address); err == nil {
		return host
	}
	return address
}
