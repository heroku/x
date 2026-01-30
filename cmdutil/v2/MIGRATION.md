# Migration Guide: cmdutil v1 → v2

## Overview

cmdutil v2 replaces logrus with log/slog and go-kit metrics with native OpenTelemetry. This is a breaking change requiring updates to all imports and some API changes.

## Import Changes

Update all imports from v1 to v2:

```go
// v1
import "github.com/heroku/x/cmdutil"
import "github.com/heroku/x/cmdutil/service"
import "github.com/heroku/x/cmdutil/svclog"

// v2
import "github.com/heroku/x/cmdutil/v2"
import "github.com/heroku/x/cmdutil/v2/service"
import "github.com/heroku/x/cmdutil/v2/svclog"
```

## Logging Changes

### Logger Type

```go
// v1
func MyFunc(logger logrus.FieldLogger) {}

// v2
func MyFunc(logger *slog.Logger) {}
```

### Logging API

```go
// v1
logger.WithFields(logrus.Fields{
    "key": "value",
    "count": 42,
}).Info("message")

logger.WithError(err).Error("failed")

// v2 - Preferred: Use typed attributes
logger.Info("message", 
    slog.String("key", "value"),
    slog.Int("count", 42))

logger.Error("failed", slog.Any("error", err))

// v2 - Also valid: Key-value pairs
logger.Info("message", "key", "value", "count", 42)
logger.Error("failed", "error", err)
```

### Structured Logging

```go
// v1
logger.WithField("user_id", id).Info("processing")

// v2 - Preferred: Use typed attributes
logger.Info("processing", slog.String("user_id", id))
logger.Info("count", slog.Int("total", 42))
logger.Error("failed", slog.Any("error", err))

// v2 - Also valid: Key-value pairs
logger.Info("processing", "user_id", id)
```

## Metrics Changes

### Metrics are Now Opt-In

**Breaking Change:** Metrics are **disabled by default** in v2.

```bash
# v1: Metrics (l2met) enabled by default
# No configuration needed

# v2: Metrics require explicit enablement
export OTEL_METRICS_ENABLED=true
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
```

**Why?** l2met was a Heroku-specific workaround that embedded metrics in logs. v2 uses proper OpenTelemetry, which requires infrastructure (OTEL collector). Metrics are opt-in to avoid requiring OTEL setup for all services.

### Metrics Setup

```go
// v1
import "github.com/heroku/x/go-kit/metrics"
import "github.com/heroku/x/go-kit/metrics/l2met"

provider := l2met.New(logger)
counter := provider.NewCounter("requests")
counter.Add(1)

// v2
import "go.opentelemetry.io/otel/metric"

// Get meter from service.Standard
meter := svc.Meter.Meter("myservice")
counter, _ := meter.Int64Counter("requests")
counter.Add(ctx, 1)
```

### Service Integration

```go
// v1
s := service.New(&cfg)
s.MetricsProvider // go-kit metrics.Provider

// v2
s := service.New(&cfg)
s.Meter // *metric.MeterProvider (nil if metrics disabled)
```

## Service Changes

### HTTP Server

```go
// v1
import "github.com/heroku/x/go-kit/metrics"

server := service.HTTP(logger, metricsProvider, handler)

// v2
server := service.HTTP(logger, handler)
```

**Removed:** `metrics.Provider` parameter - metrics are accessed via `service.Standard.Meter`

### gRPC Server

```go
// v1
server := service.GRPC(logger, grpcServer)
// Returns cmdutil.Server

// v2
handler := service.GRPCHandler(logger, grpcServer)
// Returns http.Handler

// Or for HTTP + gRPC multiplexing:
handler := service.WithGRPC(httpHandler, logger, grpcServer)
```

**Breaking Change:** gRPC now returns `http.Handler` instead of `cmdutil.Server`. This simplifies integration - single HTTP server handles both protocols on the same port.

## Removed Features

### Rollbar

Rollbar integration has been removed. Use your own error tracking solution.

```go
// v1
import "github.com/heroku/x/cmdutil/rollbar"
rollbar.Setup(logger, cfg)

// v2
// Removed - implement your own error tracking
```

### Router Bypass

TLS termination and router bypass features have been removed. Use a proxy (Envoy, nginx) for TLS termination.

```go
// v1
// HEROKU_ROUTER_* environment variables supported
// Built-in TLS termination

// v2
// Removed - use external proxy for TLS
```

## Testing Changes

### Test Logger

```go
// v1
import "github.com/heroku/x/testing/testlog"

logger, hook := testlog.New()
logger.Info("test")
hook.CheckContained(t, "test")

// v2
import "github.com/heroku/x/testing/testlog/v2"

logger, hook := testlog.New()
logger.Info("test")
hook.CheckContained(t, "test")
```

## Environment Variables

### New Variables

```bash
# Metrics (opt-in)
OTEL_METRICS_ENABLED=true                          # Enable metrics (default: false)
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318  # OTEL collector endpoint
OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf          # Protocol (default: http/protobuf)
OTEL_METRIC_EXPORT_INTERVAL=60s                    # Export interval (default: 60s)
OTEL_ENABLE_RUNTIME_METRICS=true                   # Go runtime metrics (default: false)
```

### Removed Variables

```bash
# Rollbar (removed)
ROLLBAR_TOKEN
ROLLBAR_ENV

# Router bypass (removed)
HEROKU_ROUTER_*
```

## Performance Improvements

v2 provides significant performance improvements:

- **1.7-5.5x faster logging** (slog vs logrus)
- **Zero allocations** for simple log operations
- **94% fewer allocations** for concurrent logging
- **Reduced memory pressure** - see [benchmarks](benchmarks/README.md)

## Migration Checklist

- [ ] Update all imports to v2
- [ ] Replace `logrus.FieldLogger` with `*slog.Logger`
- [ ] Update logging calls to slog API (prefer typed attributes)
- [ ] Remove `metrics.Provider` parameters
- [ ] Set `OTEL_METRICS_ENABLED=true` if metrics needed
- [ ] Configure OTEL collector endpoint
- [ ] Update gRPC integration to use handlers
- [ ] Remove rollbar setup
- [ ] Update test imports to testlog/v2
- [ ] Test thoroughly - v2 is a breaking change

## Example Migration

### Before (v1)

```go
package main

import (
    "github.com/heroku/x/cmdutil/service"
    "github.com/heroku/x/cmdutil/rollbar"
    "github.com/sirupsen/logrus"
)

func main() {
    var cfg Config
    svc := service.New(&cfg)
    
    svc.Logger.WithField("version", "1.0").Info("starting")
    
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        svc.Logger.WithField("path", r.URL.Path).Info("request")
        w.Write([]byte("OK"))
    })
    
    svc.Add(service.HTTP(svc.Logger, svc.MetricsProvider, handler))
    svc.Run()
}
```

### After (v2)

```go
package main

import (
    "log/slog"
    "github.com/heroku/x/cmdutil/v2/service"
)

func main() {
    var cfg Config
    svc := service.New(&cfg)
    
    svc.Logger.Info("starting", slog.String("version", "1.0"))
    
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        svc.Logger.Info("request", slog.String("path", r.URL.Path))
        w.Write([]byte("OK"))
    })
    
    svc.Add(service.HTTP(svc.Logger, handler))
    svc.Run()
}
```

## Getting Help

- Review the [README](README.md) for usage examples
- Check the [implementation plan](IMPLEMENTATION_PLAN.md) for design decisions
- Run [benchmarks](benchmarks/README.md) to verify performance improvements
