package limiter

import (
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

// BenchmarkAdaptiveLimiter_Allow measures the throughput of Allow() under
// the adaptive limiter.
func BenchmarkAdaptiveLimiter_Allow(b *testing.B) {
	cfg := DefaultAdaptiveConfig()
	cfg.InitialRate = float64(b.N) + 1 // ensure we never block
	cfg.MaxRate = float64(b.N) + 1
	cfg.Interval = 5 * time.Second // don't tick during benchmark
	al := NewAdaptiveLimiter(cfg)
	defer al.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			al.Allow()
		}
	})
}

// BenchmarkAdaptiveLimiter_RecordLatency measures the cost of updating the
// latency EMA from concurrent goroutines.
func BenchmarkAdaptiveLimiter_RecordLatency(b *testing.B) {
	al := NewAdaptiveLimiter(DefaultAdaptiveConfig())
	defer al.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			al.RecordLatency(5 * time.Millisecond)
		}
	})
}

// BenchmarkFixedWindowLimiter_Allow measures the throughput of Allow() for
// the fixed-window limiter.
func BenchmarkFixedWindowLimiter_Allow(b *testing.B) {
	fw := NewFixedWindowLimiter(int64(b.N)+1, 5*time.Second)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			fw.Allow()
		}
	})
}

// BenchmarkTokenBucketLimiter_Allow measures the throughput of Allow() for
// the token-bucket limiter.
func BenchmarkTokenBucketLimiter_Allow(b *testing.B) {
	rate := float64(b.N) + 1
	tbl := NewTokenBucketLimiter(rate, rate)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tbl.Allow()
		}
	})
}

// BenchmarkComparison_Sequential runs all three Allow() implementations in
// the same benchmark for a fair side-by-side comparison.
func BenchmarkComparison_Sequential(b *testing.B) {
	const rate = 1_000_000 // high enough to never throttle during benchmark

	b.Run("Adaptive", func(b *testing.B) {
		cfg := DefaultAdaptiveConfig()
		cfg.InitialRate = rate
		cfg.MaxRate = rate
		cfg.Interval = 5 * time.Second
		al := NewAdaptiveLimiter(cfg)
		defer al.Stop()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			al.Allow()
		}
	})

	b.Run("FixedWindow", func(b *testing.B) {
		fw := NewFixedWindowLimiter(rate, 5*time.Second)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			fw.Allow()
		}
	})

	b.Run("TokenBucket", func(b *testing.B) {
		tbl := NewTokenBucketLimiter(rate, rate)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			tbl.Allow()
		}
	})
}
