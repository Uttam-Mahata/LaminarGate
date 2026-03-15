package limiter

import (
	"sync/atomic"
	"time"
)

// TokenBucketLimiter is a classic token-bucket rate limiter.
//
// Tokens accumulate at a constant rate (capacity/second). Each allowed
// request consumes one token. When the bucket is empty, requests are
// rejected. The bucket is capped at capacity tokens.
//
// The implementation stores the bucket state (tokens + last-refill timestamp)
// as a single 128-bit value packed into two int64 words and updated with a
// CAS loop for lock-free operation. To keep the CAS practical we pack the
// state into one int64: tokens (upper 32 bits) + lastRefillSec (lower 32
// bits). For simplicity a mutex-free spinlock via atomic.Pointer is used
// instead.
type TokenBucketLimiter struct {
	rate     float64 // tokens per second
	capacity float64 // max tokens

	state atomic.Pointer[tbState]
}

type tbState struct {
	tokens    float64
	lastRefil time.Time
}

// NewTokenBucketLimiter creates a TokenBucketLimiter with the given rate
// (tokens/second) and bucket capacity. The bucket starts full.
func NewTokenBucketLimiter(rate, capacity float64) *TokenBucketLimiter {
	tbl := &TokenBucketLimiter{rate: rate, capacity: capacity}
	initial := &tbState{tokens: capacity, lastRefil: time.Now()}
	tbl.state.Store(initial)
	return tbl
}

// Allow returns true and consumes one token if available, false otherwise.
func (tbl *TokenBucketLimiter) Allow() bool {
	now := time.Now()
	for {
		old := tbl.state.Load()
		elapsed := now.Sub(old.lastRefil).Seconds()
		newTokens := old.tokens + elapsed*tbl.rate
		if newTokens > tbl.capacity {
			newTokens = tbl.capacity
		}
		if newTokens < 1 {
			return false
		}
		next := &tbState{tokens: newTokens - 1, lastRefil: now}
		if tbl.state.CompareAndSwap(old, next) {
			return true
		}
		// Another goroutine updated concurrently; retry.
	}
}
