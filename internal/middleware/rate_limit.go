package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/fekuna/omnipos-gateway/config"
	"github.com/fekuna/omnipos-pkg/cache"
	"github.com/fekuna/omnipos-pkg/logger"
	"github.com/go-redis/redis_rate/v10"
	"go.uber.org/zap"
)

type RateLimiter struct {
	limiter *redis_rate.Limiter
	cfg     config.RateLimitConfig
	logger  logger.ZapLogger
}

func NewRateLimiter(redisClient *cache.RedisClient, cfg config.RateLimitConfig, log logger.ZapLogger) *RateLimiter {
	return &RateLimiter{
		limiter: redis_rate.NewLimiter(redisClient.Client),
		cfg:     cfg,
		logger:  log,
	}
}

func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.cfg.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		ctx := r.Context()
		key, limit := rl.getLimit(r)

		res, err := rl.limiter.Allow(ctx, key, limit)
		if err != nil {
			rl.logger.Error("rate limit error", zap.Error(err))
			// Fail open or closed? Here we fail open to avoid blocking valid traffic on redis errors
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit.Rate))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", res.Remaining))
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", res.ResetAfter/time.Millisecond))

		if res.Allowed == 0 {
			w.Header().Set("Retry-After", fmt.Sprintf("%d", res.RetryAfter/time.Second))
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) getLimit(r *http.Request) (string, redis_rate.Limit) {
	// Check for Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		// Use a simple hash of the auth header or just the header itself as key
		// In a real scenario, we might want to extract the user ID here,
		// but since we are upstream of the auth middleware, we might not have it parsed yet unless we duplicate logic.
		// However, the main.go sets up auth middleware *after* standard HTTP middleware typically?
		// Actually, in main.go, the generic mux is wrapped.
		// To be safe and efficient, we'll use the auth header as the key if present.
		// We can prefix it to avoid collisions.
		key := fmt.Sprintf("rate_limit:auth:%s", authHeader)
		return key, redis_rate.Limit{
			Rate:   rl.cfg.AuthRPS,
			Burst:  rl.cfg.AuthBurst,
			Period: time.Second,
		}
	}

	// Fallback to IP
	ip := getClientIP(r)
	key := fmt.Sprintf("rate_limit:ip:%s", ip)
	return key, redis_rate.Limit{
		Rate:   rl.cfg.PublicRPS,
		Burst:  rl.cfg.PublicBurst,
		Period: time.Second,
	}
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}

	// Check X-Real-IP
	xrp := r.Header.Get("X-Real-IP")
	if xrp != "" {
		return xrp
	}

	// Fallback to RemoteAddr
	// RemoteAddr contains port, need to strip it
	addr := r.RemoteAddr
	if strings.Contains(addr, ":") {
		// handle ipv6 [::1]:port or ipv4 1.2.3.4:port
		// simple split by last colon
		lastColon := strings.LastIndex(addr, ":")
		if lastColon != -1 {
			addr = addr[:lastColon]
		}
	}
	return addr
}
