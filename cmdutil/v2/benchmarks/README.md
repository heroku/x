# cmdutil/v2 Benchmarks

Performance comparison between cmdutil v1 (logrus) and v2 (slog).

## Running Benchmarks

```bash
# Run v1 benchmarks
cd cmdutil/benchmarks
go test -bench=. -benchmem

# Run v2 benchmarks
cd cmdutil/v2/benchmarks
go test -bench=. -benchmem
```

## Results (Apple M1 Pro)

### Structured Logging (4 fields)

| Version | Time/op | Speedup | Bytes/op | Allocs/op |
|---------|---------|---------|----------|-----------|
| v1 (logrus) | 1491 ns | 1x | 1009 B | 20 |
| v2 (slog) | 803 ns | **1.9x faster** | 192 B | 4 |

**Improvement:** 1.9x faster, 81% less memory, 80% fewer allocations

### Simple Logging (no fields)

| Version | Time/op | Speedup | Bytes/op | Allocs/op |
|---------|---------|---------|----------|-----------|
| v1 (logrus) | 779 ns | 1x | 408 B | 14 |
| v2 (slog) | 450 ns | **1.7x faster** | 0 B | 0 |

**Improvement:** 1.7x faster, **zero allocations**

### With Context (pre-configured fields)

| Version | Time/op | Speedup | Bytes/op | Allocs/op |
|---------|---------|---------|----------|-----------|
| v1 (logrus) | 1196 ns | 1x | 869 B | 16 |
| v2 (slog) | 480 ns | **2.5x faster** | 48 B | 1 |

**Improvement:** 2.5x faster, 94% less memory, 94% fewer allocations

### Concurrent Logging

| Version | Time/op | Speedup | Bytes/op | Allocs/op |
|---------|---------|---------|----------|-----------|
| v1 (logrus) | 1368 ns | 1x | 854 B | 17 |
| v2 (slog) | 249 ns | **5.5x faster** | 48 B | 1 |

**Improvement:** 5.5x faster, 94% less memory, 94% fewer allocations

## Memory Savings

For a service logging 10,000 requests/second:

**v1 (logrus):**
- Structured: 1009 B × 10,000 = ~10 MB/s
- Simple: 408 B × 10,000 = ~4 MB/s

**v2 (slog):**
- Structured: 192 B × 10,000 = ~2 MB/s
- Simple: 0 B × 10,000 = **0 MB/s**

**Savings:** 8 MB/s (structured) or 4 MB/s (simple)

## Key Takeaways

1. **slog is consistently faster** - 1.7-5.5x speedup depending on scenario
2. **Concurrent logging is 5.5x faster** - huge benefit for high-throughput services
3. **Zero allocations for simple logs** - massive GC benefit
4. **Context logging is 2.5x faster** - common pattern in real services
5. **80-94% fewer allocations** across all scenarios

## Notes

- Benchmarks run with output discarded (`io.Discard`) to measure pure logging overhead
- Real-world performance may vary based on output destination (stdout, file, network)
- slog's performance advantage increases with higher log volumes due to reduced GC pressure
