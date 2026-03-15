// Package limiter provides an adaptive, latency-aware rate limiter and
// classic rate limiters for comparison.
//
// # Adaptive Limiter
//
// The adaptive limiter treats the allowed request rate R(t) as a continuous
// function governed by a PD (Proportional–Derivative) controller:
//
//	dR/dt = Kp·e(t) + Kd·de(t)/dt
//
// where e(t) = targetLatency − actualLatency.
//
// In discrete time (Euler's method):
//
//	R_{n+1} = R_n + [Kp·e_n + Kd·(e_n − e_{n-1})/Δt]·Δt
//
// R is clamped to [1, MaxRate] to prevent system black-out or unbounded
// growth.
//
// Three components operate independently:
//
//   - Monitor   – records per-request latency and maintains an exponential
//     moving average (EMA) using lock-free atomics.
//   - Controller – a background goroutine that ticks every Interval and
//     applies the PD law to update the global rate.
//   - Enforcer  – the hot path: an atomic counter that resets each Interval;
//     Allow() returns false when the counter exceeds the current rate.
package limiter

import (
	"math"
	"sync/atomic"
	"time"
)

// AdaptiveLimiter is a latency-feedback PD-controller rate limiter.
// All hot-path operations are lock-free.
type AdaptiveLimiter struct {
	// configuration (read-only after construction)
	targetLatency time.Duration
	maxRate       float64
	kp            float64
	kd            float64
	interval      time.Duration

	// Monitor – latency tracking (EMA stored as nanoseconds in int64)
	latencyEMA atomic.Int64 // nanoseconds
	emaAlpha   float64      // smoothing factor ∈ (0,1]

	// Controller state (written only by the controller goroutine)
	// currentRate is the authoritative float64 state for the PD accumulator.
	// rateAtomic stores the same value bit-for-bit (via math.Float64bits) for
	// lock-free reads from Allow() and CurrentRate().
	currentRate float64
	rateAtomic  atomic.Uint64 // stores math.Float64bits(currentRate)
	prevError   float64

	// Enforcer – per-interval request counter
	counter  atomic.Int64
	windowID atomic.Int64 // monotonically increasing window identifier

	done chan struct{}
}

// AdaptiveConfig holds the tuning parameters for AdaptiveLimiter.
type AdaptiveConfig struct {
	// TargetLatency is the desired p50/average latency threshold.
	TargetLatency time.Duration
	// MaxRate is the ceiling for the allowed request rate (req/s).
	MaxRate float64
	// InitialRate is the starting allowed request rate (req/s).
	InitialRate float64
	// Kp is the proportional gain.
	Kp float64
	// Kd is the derivative gain.
	Kd float64
	// Interval is the controller tick duration (e.g. 100 ms).
	Interval time.Duration
	// EMAAlpha is the exponential moving-average smoothing factor (0,1].
	// A value of 1.0 means no smoothing (last sample only).
	EMAAlpha float64
}

// DefaultAdaptiveConfig returns a sensibly tuned configuration.
func DefaultAdaptiveConfig() AdaptiveConfig {
	return AdaptiveConfig{
		TargetLatency: 10 * time.Millisecond,
		MaxRate:       1000,
		InitialRate:   500,
		Kp:            0.5,
		Kd:            0.1,
		Interval:      100 * time.Millisecond,
		EMAAlpha:      0.2,
	}
}

// NewAdaptiveLimiter constructs and starts an AdaptiveLimiter with the
// given configuration. Call Stop() to release the background goroutine.
func NewAdaptiveLimiter(cfg AdaptiveConfig) *AdaptiveLimiter {
	if cfg.EMAAlpha <= 0 || cfg.EMAAlpha > 1 {
		cfg.EMAAlpha = 0.2
	}
	if cfg.InitialRate <= 0 {
		cfg.InitialRate = 1
	}
	if cfg.MaxRate <= 0 {
		cfg.MaxRate = 1000
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 100 * time.Millisecond
	}

	al := &AdaptiveLimiter{
		targetLatency: cfg.TargetLatency,
		maxRate:       cfg.MaxRate,
		kp:            cfg.Kp,
		kd:            cfg.Kd,
		interval:      cfg.Interval,
		emaAlpha:      cfg.EMAAlpha,
		currentRate:   cfg.InitialRate,
		done:          make(chan struct{}),
	}
	al.rateAtomic.Store(math.Float64bits(cfg.InitialRate))
	al.latencyEMA.Store(int64(cfg.TargetLatency)) // start at target

	go al.controlLoop()
	return al
}

// RecordLatency informs the Monitor about the observed latency of a request.
// It is safe to call from multiple goroutines concurrently.
func (al *AdaptiveLimiter) RecordLatency(d time.Duration) {
	ns := d.Nanoseconds()
	for {
		old := al.latencyEMA.Load()
		// EMA: new = alpha*sample + (1-alpha)*old
		newEMA := int64(al.emaAlpha*float64(ns) + (1-al.emaAlpha)*float64(old))
		if al.latencyEMA.CompareAndSwap(old, newEMA) {
			return
		}
	}
}

// CurrentLatency returns the current EMA-smoothed latency estimate.
func (al *AdaptiveLimiter) CurrentLatency() time.Duration {
	return time.Duration(al.latencyEMA.Load())
}

// CurrentRate returns the current allowed request rate (req/s).
func (al *AdaptiveLimiter) CurrentRate() float64 {
	return math.Float64frombits(al.rateAtomic.Load())
}

// Allow is the Enforcer hot path. It returns true if the request is within
// the current rate limit for this interval window, false otherwise.
// O(1), lock-free.
func (al *AdaptiveLimiter) Allow() bool {
	rate := math.Float64frombits(al.rateAtomic.Load())
	count := al.counter.Add(1)
	return float64(count) <= rate
}

// Stop terminates the background controller goroutine.
func (al *AdaptiveLimiter) Stop() {
	close(al.done)
}

// controlLoop is the background Controller goroutine.
func (al *AdaptiveLimiter) controlLoop() {
	ticker := time.NewTicker(al.interval)
	defer ticker.Stop()

	dt := al.interval.Seconds()

	for {
		select {
		case <-al.done:
			return
		case <-ticker.C:
			al.tick(dt)
		}
	}
}

// tick applies the discrete PD law for one time step Δt (in seconds).
func (al *AdaptiveLimiter) tick(dt float64) {
	// Reset the per-interval enforcer counter for the new window.
	al.counter.Store(0)

	// Compute current error: e = target − actual (in seconds for unit consistency).
	actualLatency := time.Duration(al.latencyEMA.Load())
	e := (al.targetLatency - actualLatency).Seconds()

	// Derivative of error: de/dt ≈ (e_n − e_{n-1}) / Δt
	dedt := (e - al.prevError) / dt
	al.prevError = e

	// PD update: R_{n+1} = R_n + [Kp·e + Kd·(de/dt)]·Δt
	delta := (al.kp*e + al.kd*dedt) * dt
	newRate := al.currentRate + delta

	// Clamp to [1, MaxRate]
	newRate = math.Max(1, math.Min(al.maxRate, newRate))
	al.currentRate = newRate
	al.rateAtomic.Store(math.Float64bits(newRate))
}
