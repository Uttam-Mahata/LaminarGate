package limiter

import (
	"sync/atomic"
	"time"
)

// FixedWindowLimiter is a classic fixed-window counter rate limiter.
//
// Requests are counted in fixed-duration windows (e.g. 1 second). When the
// count reaches the limit, further requests in the same window are rejected.
// At the start of each new window the counter resets atomically.
//
// This implementation uses lock-free atomics for the hot path.
type FixedWindowLimiter struct {
	limit    int64
	window   time.Duration
	counter  atomic.Int64
	windowID atomic.Int64 // = UnixNano / window.Nanoseconds()
}

// NewFixedWindowLimiter creates a FixedWindowLimiter that allows at most
// limit requests per window duration.
func NewFixedWindowLimiter(limit int64, window time.Duration) *FixedWindowLimiter {
	return &FixedWindowLimiter{
		limit:  limit,
		window: window,
	}
}

// Allow returns true if the request falls within the current window quota.
func (fw *FixedWindowLimiter) Allow() bool {
	now := time.Now().UnixNano()
	currentWindow := now / fw.window.Nanoseconds()

	// If we are in a new window, reset the counter.
	for {
		prev := fw.windowID.Load()
		if prev == currentWindow {
			break
		}
		if fw.windowID.CompareAndSwap(prev, currentWindow) {
			fw.counter.Store(0)
			break
		}
	}

	return fw.counter.Add(1) <= fw.limit
}
