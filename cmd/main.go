// Command laminardemo runs a simulation that demonstrates the adaptive
// limiter responding to changing latency conditions alongside classic
// fixed-window and token-bucket limiters.
package main

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Uttam-Mahata/LaminarGate/limiter"
)

func main() {
	fmt.Println("=== LaminarGate – Adaptive Rate Limiter Demo ===")
	fmt.Println()

	runAdaptiveSimulation()
	fmt.Println()
	runComparison()
}

// runAdaptiveSimulation shows the adaptive limiter tracking latency changes.
func runAdaptiveSimulation() {
	fmt.Println("--- Adaptive Limiter: Latency-Response Simulation ---")

	cfg := limiter.DefaultAdaptiveConfig()
	cfg.TargetLatency = 10 * time.Millisecond
	cfg.MaxRate = 1000
	cfg.InitialRate = 200
	cfg.Kp = 5
	cfg.Kd = 0.5
	cfg.Interval = 200 * time.Millisecond
	cfg.EMAAlpha = 0.3

	al := limiter.NewAdaptiveLimiter(cfg)
	defer al.Stop()

	type phase struct {
		label   string
		latency time.Duration
		dur     time.Duration
	}

	phases := []phase{
		{"Normal (5 ms)",    5 * time.Millisecond, 800 * time.Millisecond},
		{"Spike (50 ms)",   50 * time.Millisecond, 800 * time.Millisecond},
		{"Recovery (8 ms)",  8 * time.Millisecond, 800 * time.Millisecond},
	}

	for _, p := range phases {
		fmt.Printf("  Phase: %-22s", p.label)
		deadline := time.Now().Add(p.dur)
		for time.Now().Before(deadline) {
			// Simulate a request with jitter.
			jitter := time.Duration(rand.Int63n(int64(p.latency / 2)))
			al.RecordLatency(p.latency + jitter)
			time.Sleep(1 * time.Millisecond)
		}
		fmt.Printf("rate=%.0f req/s  latency_ema=%v\n",
			al.CurrentRate(), al.CurrentLatency().Round(time.Microsecond))
	}
}

// runComparison sends a fixed load through all three limiters and reports
// the acceptance ratios.
func runComparison() {
	fmt.Println("--- Comparison: Adaptive vs Fixed-Window vs Token-Bucket ---")
	fmt.Printf("  %-16s  %8s  %8s  %8s\n", "Limiter", "Sent", "Allowed", "Rate%")

	const (
		rateLimit  = 200             // req per window / tokens per second
		numWorkers = 20
		testDur    = 500 * time.Millisecond
	)

	type result struct {
		name    string
		sent    int64
		allowed int64
	}

	run := func(name string, allowFn func() bool) result {
		var sent, allowed atomic.Int64
		var wg sync.WaitGroup
		stop := make(chan struct{})

		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					select {
					case <-stop:
						return
					default:
						sent.Add(1)
						if allowFn() {
							allowed.Add(1)
						}
						time.Sleep(time.Millisecond)
					}
				}
			}()
		}

		time.Sleep(testDur)
		close(stop)
		wg.Wait()
		return result{name, sent.Load(), allowed.Load()}
	}

	// Adaptive
	cfg := limiter.DefaultAdaptiveConfig()
	cfg.InitialRate = rateLimit
	cfg.MaxRate = rateLimit * 2
	cfg.TargetLatency = 5 * time.Millisecond
	cfg.Interval = 100 * time.Millisecond
	al := limiter.NewAdaptiveLimiter(cfg)
	for i := 0; i < 100; i++ {
		al.RecordLatency(5 * time.Millisecond) // stable latency
	}
	resAdaptive := run("Adaptive", al.Allow)
	al.Stop()

	// Fixed Window
	fw := limiter.NewFixedWindowLimiter(rateLimit, time.Second)
	resFW := run("FixedWindow", fw.Allow)

	// Token Bucket
	tbl := limiter.NewTokenBucketLimiter(rateLimit, rateLimit)
	resTB := run("TokenBucket", tbl.Allow)

	for _, r := range []result{resAdaptive, resFW, resTB} {
		pct := 100.0 * float64(r.allowed) / float64(r.sent)
		fmt.Printf("  %-16s  %8d  %8d  %7.1f%%\n", r.name, r.sent, r.allowed, pct)
	}
}
