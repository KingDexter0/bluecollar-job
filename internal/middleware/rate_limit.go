package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateLimitBucket struct {
	windowStart time.Time
	count       int
}

type RateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	buckets map[string]rateLimitBucket
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	if limit <= 0 {
		limit = 60
	}
	if window <= 0 {
		window = time.Minute
	}
	return &RateLimiter{
		limit:   limit,
		window:  window,
		buckets: map[string]rateLimitBucket{},
	}
}

func (r *RateLimiter) Middleware(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := name + ":" + c.ClientIP()
		if !r.allow(key, time.Now()) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, ErrorResponse{Error: ErrorBody{
				Code:    "rate_limited",
				Message: "too many requests, please retry later",
			}})
			return
		}
		c.Next()
	}
}

func (r *RateLimiter) allow(key string, now time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	bucket := r.buckets[key]
	if bucket.windowStart.IsZero() || now.Sub(bucket.windowStart) >= r.window {
		r.buckets[key] = rateLimitBucket{windowStart: now, count: 1}
		r.prune(now)
		return true
	}
	if bucket.count >= r.limit {
		return false
	}
	bucket.count++
	r.buckets[key] = bucket
	return true
}

func (r *RateLimiter) prune(now time.Time) {
	for key, bucket := range r.buckets {
		if now.Sub(bucket.windowStart) > 2*r.window {
			delete(r.buckets, key)
		}
	}
}
