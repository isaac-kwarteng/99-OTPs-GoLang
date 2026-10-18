package main

import (
	"context"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type rateWindow struct {
	started time.Time
	count   int
}

type rateLimiter struct {
	mu      sync.Mutex
	windows map[string]rateWindow
	limit   map[string]int
	lastCleanup time.Time
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{
		windows: make(map[string]rateWindow),
		limit: map[string]int{
			"/" + otpServiceName + "/SendOTP":   5,
			"/" + otpServiceName + "/VerifyOTP": 10,
		},
	}
}

func (limiter *rateLimiter) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, request any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		limit, applies := limiter.limit[info.FullMethod]
		if applies && !limiter.allow(clientIP(ctx), info.FullMethod, limit, time.Now()) {
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}
		return handler(ctx, request)
	}
}

func (limiter *rateLimiter) allow(ip, method string, limit int, now time.Time) bool {
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

	key := method + ":" + ip
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

func clientIP(ctx context.Context) string {
	remote, exists := peer.FromContext(ctx)
	if !exists || remote.Addr == nil {
		return "unknown"
	}
	if host, _, err := net.SplitHostPort(remote.Addr.String()); err == nil {
		return host
	}
	return remote.Addr.String()
}