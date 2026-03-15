# LaminarGate Benchmarks

This document contains performance benchmarks for the different rate limiters implemented in LaminarGate, run on an Intel(R) Xeon(R) Processor @ 2.30GHz.

## Benchmark Results

```text
goos: linux
goarch: amd64
pkg: github.com/Uttam-Mahata/LaminarGate/limiter
cpu: Intel(R) Xeon(R) Processor @ 2.30GHz
BenchmarkAdaptiveLimiter_Allow-4           	57482576	        48.72 ns/op	       0 B/op	       0 allocs/op
BenchmarkAdaptiveLimiter_RecordLatency-4   	17797700	        67.19 ns/op	       0 B/op	       0 allocs/op
BenchmarkFixedWindowLimiter_Allow-4        	14413356	        81.98 ns/op	       0 B/op	       0 allocs/op
BenchmarkTokenBucketLimiter_Allow-4        	 4011882	       303.3 ns/op	     100 B/op	       3 allocs/op
BenchmarkComparison_Sequential/Adaptive-4  	122174253	         9.672 ns/op	       0 B/op	       0 allocs/op
BenchmarkComparison_Sequential/FixedWindow-4         	14908682	        80.81 ns/op	       0 B/op	       0 allocs/op
BenchmarkComparison_Sequential/TokenBucket-4         	12811122	        92.00 ns/op	       5 B/op	       0 allocs/op
```

## Analysis

*   **Adaptive Limiter**: The adaptive limiter showcases exceptional performance for `Allow()` (under 10 ns/op in sequential, ~48 ns/op in parallel), making it the fastest option in both sequential and parallel environments. This highlights the efficiency of its lock-free hot path using `atomic.Uint64`. `RecordLatency()` also performs very well (~67 ns/op).
*   **Fixed Window Limiter**: The Fixed Window limiter provides robust performance (~81 ns/op in both sequential and parallel operations) with zero memory allocations, demonstrating a highly optimized lock-free design using `atomic.Int64`.
*   **Token Bucket Limiter**: The Token Bucket limiter is the slowest among the three (~92 ns/op sequentially, ~303 ns/op parallel) and requires allocations. The CAS loop logic under heavy concurrency leads to increased overhead compared to the other methods.

## Conclusion

The **Adaptive Limiter** not only offers dynamic adjustments based on latency but also boasts the best raw performance characteristics of the tested algorithms, particularly excelling in high-throughput lock-free validation (`Allow()`).
