# LaminarGate
This project implements an **Adaptive Limiter** that treats the system's "Request Rate" as a continuous function $R(t)$ and adjusts it dynamically based on real-time latency feedback.

## Benchmarks

LaminarGate has been benchmarked against traditional rate limiting algorithms like Fixed Window and Token Bucket.

The Adaptive Limiter is not only dynamic but highly performant, providing lock-free `Allow()` checks in under 50 ns/op (highly concurrent) and ~10 ns/op sequentially with 0 memory allocations.

For full benchmark results and comparisons, see [benchmark.md](benchmark.md).
