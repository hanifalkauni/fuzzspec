package fuzzer

import (
	"context"

	"golang.org/x/time/rate"
)

// RateLimiter wraps token-bucket rate limiting.
type RateLimiter struct {
	limiter *rate.Limiter
}

// NewRateLimiter creates a RateLimiter with a given requests-per-second limit.
func NewRateLimiter(rps int) *RateLimiter {
	if rps <= 0 {
		rps = 20
	}
	// Burst equals RPS or minimum 5 to allow initial worker startup
	burst := rps
	if burst < 5 {
		burst = 5
	}
	return &RateLimiter{
		limiter: rate.NewLimiter(rate.Limit(rps), burst),
	}
}

// Wait blocks until the rate limiter allows a request or the context is cancelled.
func (r *RateLimiter) Wait(ctx context.Context) error {
	return r.limiter.Wait(ctx)
}
