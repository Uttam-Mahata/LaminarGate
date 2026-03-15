package main

import "C"
import (
	"sync"
	"time"

	"github.com/Uttam-Mahata/LaminarGate/limiter"
)

var (
	adaptiveMu        sync.RWMutex
	adaptiveLimiters  = make(map[int64]*limiter.AdaptiveLimiter)
	adaptiveLimiterID int64
)

//export CreateAdaptiveLimiter
func CreateAdaptiveLimiter(targetLatencyMs, maxRate, initialRate, kp, kd float64, intervalMs float64, emaAlpha float64) int64 {
	cfg := limiter.DefaultAdaptiveConfig()
	if targetLatencyMs > 0 {
		cfg.TargetLatency = time.Duration(targetLatencyMs) * time.Millisecond
	}
	if maxRate > 0 {
		cfg.MaxRate = maxRate
	}
	if initialRate > 0 {
		cfg.InitialRate = initialRate
	}
	if kp > 0 {
		cfg.Kp = kp
	}
	if kd > 0 {
		cfg.Kd = kd
	}
	if intervalMs > 0 {
		cfg.Interval = time.Duration(intervalMs) * time.Millisecond
	}
	if emaAlpha > 0 {
		cfg.EMAAlpha = emaAlpha
	}

	al := limiter.NewAdaptiveLimiter(cfg)

	adaptiveMu.Lock()
	defer adaptiveMu.Unlock()
	adaptiveLimiterID++
	id := adaptiveLimiterID
	adaptiveLimiters[id] = al
	return id
}

//export AdaptiveLimiterAllow
func AdaptiveLimiterAllow(id int64) bool {
	adaptiveMu.RLock()
	al, ok := adaptiveLimiters[id]
	adaptiveMu.RUnlock()
	if !ok {
		return false
	}
	return al.Allow()
}

//export AdaptiveLimiterRecordLatency
func AdaptiveLimiterRecordLatency(id int64, latencyMs float64) {
	adaptiveMu.RLock()
	al, ok := adaptiveLimiters[id]
	adaptiveMu.RUnlock()
	if !ok {
		return
	}
	al.RecordLatency(time.Duration(latencyMs * float64(time.Millisecond)))
}

//export AdaptiveLimiterCurrentRate
func AdaptiveLimiterCurrentRate(id int64) float64 {
	adaptiveMu.RLock()
	al, ok := adaptiveLimiters[id]
	adaptiveMu.RUnlock()
	if !ok {
		return 0.0
	}
	return al.CurrentRate()
}

//export AdaptiveLimiterStop
func AdaptiveLimiterStop(id int64) {
	adaptiveMu.Lock()
	defer adaptiveMu.Unlock()
	al, ok := adaptiveLimiters[id]
	if ok {
		al.Stop()
		delete(adaptiveLimiters, id)
	}
}

// A main function is required for cgo
func main() {}
