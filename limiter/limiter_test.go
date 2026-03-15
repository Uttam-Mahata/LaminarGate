package limiter

import (
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// AdaptiveLimiter tests
// ---------------------------------------------------------------------------

func TestAdaptiveLimiter_Allow_WithinRate(t *testing.T) {
	cfg := DefaultAdaptiveConfig()
	cfg.InitialRate = 10
	cfg.Interval = 500 * time.Millisecond
	al := NewAdaptiveLimiter(cfg)
	defer al.Stop()

	allowed := 0
	for i := 0; i < 10; i++ {
		if al.Allow() {
			allowed++
		}
	}
	if allowed != 10 {
		t.Errorf("expected 10 allowed, got %d", allowed)
	}
}

func TestAdaptiveLimiter_Allow_ExceedsRate(t *testing.T) {
	cfg := DefaultAdaptiveConfig()
	cfg.InitialRate = 5
	cfg.Interval = 500 * time.Millisecond
	al := NewAdaptiveLimiter(cfg)
	defer al.Stop()

	allowed := 0
	for i := 0; i < 20; i++ {
		if al.Allow() {
			allowed++
		}
	}
	// Only the first 5 should be allowed in this window.
	if allowed != 5 {
		t.Errorf("expected 5 allowed, got %d", allowed)
	}
}

func TestAdaptiveLimiter_RecordLatency_ReducesRate(t *testing.T) {
	cfg := DefaultAdaptiveConfig()
	cfg.InitialRate = 100
	cfg.MaxRate = 200
	cfg.TargetLatency = 10 * time.Millisecond
	cfg.Kp = 10
	cfg.Kd = 1
	cfg.Interval = 50 * time.Millisecond
	cfg.EMAAlpha = 1.0 // no smoothing; react immediately

	al := NewAdaptiveLimiter(cfg)
	defer al.Stop()

	// Simulate high latency (10× the target).
	for i := 0; i < 50; i++ {
		al.RecordLatency(100 * time.Millisecond)
	}

	time.Sleep(200 * time.Millisecond) // let the controller run several ticks

	rate := al.CurrentRate()
	if rate >= 100 {
		t.Errorf("expected rate to drop below 100 under high latency, got %.2f", rate)
	}
}

func TestAdaptiveLimiter_RecordLatency_IncreasesRate(t *testing.T) {
	cfg := DefaultAdaptiveConfig()
	cfg.InitialRate = 50
	cfg.MaxRate = 500
	cfg.TargetLatency = 50 * time.Millisecond
	cfg.Kp = 500  // aggressive proportional gain so rate visibly increases
	cfg.Kd = 0
	cfg.Interval = 50 * time.Millisecond
	cfg.EMAAlpha = 1.0

	al := NewAdaptiveLimiter(cfg)
	defer al.Stop()

	// Simulate very low latency (well below target).
	for i := 0; i < 50; i++ {
		al.RecordLatency(1 * time.Millisecond)
	}

	time.Sleep(200 * time.Millisecond)

	rate := al.CurrentRate()
	if rate <= 50 {
		t.Errorf("expected rate to rise above 50 under low latency, got %.2f", rate)
	}
}

func TestAdaptiveLimiter_RateClamped(t *testing.T) {
	cfg := DefaultAdaptiveConfig()
	cfg.InitialRate = 3
	cfg.MaxRate = 10
	cfg.TargetLatency = 5 * time.Millisecond
	cfg.Kp = 100
	cfg.Kd = 0
	cfg.Interval = 50 * time.Millisecond
	cfg.EMAAlpha = 1.0

	al := NewAdaptiveLimiter(cfg)
	defer al.Stop()

	// Force latency to zero → rate should want to go to MaxRate.
	for i := 0; i < 50; i++ {
		al.RecordLatency(0)
	}
	time.Sleep(300 * time.Millisecond)

	rate := al.CurrentRate()
	if rate > cfg.MaxRate {
		t.Errorf("rate %.2f exceeded MaxRate %.2f", rate, cfg.MaxRate)
	}
	if rate < 1 {
		t.Errorf("rate %.2f is below minimum of 1", rate)
	}
}

func TestAdaptiveLimiter_CurrentLatency(t *testing.T) {
	cfg := DefaultAdaptiveConfig()
	cfg.EMAAlpha = 1.0
	al := NewAdaptiveLimiter(cfg)
	defer al.Stop()

	al.RecordLatency(20 * time.Millisecond)
	got := al.CurrentLatency()
	if got != 20*time.Millisecond {
		t.Errorf("expected 20ms, got %v", got)
	}
}

// ---------------------------------------------------------------------------
// FixedWindowLimiter tests
// ---------------------------------------------------------------------------

func TestFixedWindowLimiter_Allow_WithinLimit(t *testing.T) {
	fw := NewFixedWindowLimiter(5, time.Second)

	allowed := 0
	for i := 0; i < 5; i++ {
		if fw.Allow() {
			allowed++
		}
	}
	if allowed != 5 {
		t.Errorf("expected 5 allowed, got %d", allowed)
	}
}

func TestFixedWindowLimiter_Allow_ExceedsLimit(t *testing.T) {
	fw := NewFixedWindowLimiter(3, time.Second)

	allowed := 0
	for i := 0; i < 10; i++ {
		if fw.Allow() {
			allowed++
		}
	}
	if allowed != 3 {
		t.Errorf("expected 3 allowed, got %d", allowed)
	}
}

func TestFixedWindowLimiter_ResetsAfterWindow(t *testing.T) {
	fw := NewFixedWindowLimiter(2, 50*time.Millisecond)

	// Exhaust first window.
	fw.Allow()
	fw.Allow()
	if fw.Allow() {
		t.Error("third request in first window should have been denied")
	}

	// Wait for the window to roll over.
	time.Sleep(60 * time.Millisecond)

	// New window should allow again.
	if !fw.Allow() {
		t.Error("first request in second window should be allowed")
	}
}

// ---------------------------------------------------------------------------
// TokenBucketLimiter tests
// ---------------------------------------------------------------------------

func TestTokenBucketLimiter_Allow_WithinCapacity(t *testing.T) {
	tbl := NewTokenBucketLimiter(10, 5)

	allowed := 0
	for i := 0; i < 5; i++ {
		if tbl.Allow() {
			allowed++
		}
	}
	if allowed != 5 {
		t.Errorf("expected 5 allowed, got %d", allowed)
	}
}

func TestTokenBucketLimiter_Allow_ExceedsCapacity(t *testing.T) {
	tbl := NewTokenBucketLimiter(10, 3)

	allowed := 0
	for i := 0; i < 10; i++ {
		if tbl.Allow() {
			allowed++
		}
	}
	if allowed != 3 {
		t.Errorf("expected 3 allowed (full bucket), got %d", allowed)
	}
}

func TestTokenBucketLimiter_RefillsOverTime(t *testing.T) {
	// rate=20 tokens/s, capacity=5 → refills 1 token in 50 ms
	tbl := NewTokenBucketLimiter(20, 5)

	// Drain the bucket.
	for i := 0; i < 5; i++ {
		tbl.Allow()
	}
	if tbl.Allow() {
		t.Error("bucket should be empty immediately after draining")
	}

	// Wait for ~1 token to refill (≥50 ms).
	time.Sleep(60 * time.Millisecond)
	if !tbl.Allow() {
		t.Error("expected at least one token to have refilled after 60 ms")
	}
}
