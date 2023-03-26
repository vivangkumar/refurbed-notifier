package ratelimiter

import (
	"sync"
	"time"
)

// RateLimiter represents a simple token based rate limiter.
//
// It limits based on the number of tokens it currently has available.
//
// A token is used up when a single request is made.
// Tokens are refilled based on the number of requests allowed
// per second.
//
// If we can make 4 request per second and we use up one token,
// it is refilled again after 250ms.
type RateLimiter struct {
	// tokens represents the number of tokens that the rat
	// limiter currently has.
	tokens uint64

	// max represents the max allowed tokens.
	max uint64

	// refillTokens specifies the number of tokens that the rate
	// limiter is refilled with (every duration).
	refillTokens uint64

	// refillEvery is the duration after which
	// tokens are refilled.
	refillEvery time.Duration

	m    sync.Mutex
	stop chan struct{}
}

// New constructs a rate limiter that accepts the max requests
// allowed per second along with the number of tokens that the
// rate limiter is refilled with after passing of the duration
// determined by the rps.
func New(rps uint64, refill uint64) *RateLimiter {
	r := &RateLimiter{
		tokens:       rps,
		max:          rps,
		refillTokens: refill,
		refillEvery:  time.Duration(float64(time.Second) / float64(rps)),
		stop:         make(chan struct{}),
	}

	return r
}

// Start is responsible for refilling the tokens
// every refill duration.
//
// There is a risk that we might allow a few tokens
// to be acquired even after stopping the rate limiter
// due to pseudo random nature of channels (i.e when both
// conditions are satisfied)
//
// Note that Stop must be called to gracefully exit.
func (r *RateLimiter) Start() {
	go func() {
		t := time.NewTicker(r.refillEvery)
		defer t.Stop()

		for {
			select {
			case <-t.C:
				r.m.Lock()
				t := r.tokens + r.refillTokens
				if t > r.max {
					t = r.max
				}
				r.tokens = t
				r.m.Unlock()
			case <-r.stop:
				return
			}
		}
	}()
}

// Add attempts to take a single token from the rate limiter.
//
// If there are tokens available, it returns true.
// Otherwise, the method returns false, indicating that we
// have reached the rate limit.
func (r *RateLimiter) Add() bool {
	r.m.Lock()
	defer r.m.Unlock()

	if r.tokens > 0 {
		r.tokens--
		return true
	}

	return false
}

// Stop gracefully stops the rate limiter.
func (r *RateLimiter) Stop() {
	close(r.stop)
}
