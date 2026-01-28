# cmdutil/v2

Service utilities using log/slog and native OpenTelemetry.

## Changes from v1

- Replaces logrus with log/slog
- Replaces go-kit metrics with native OTEL SDK
- Removes rollbar package

## Usage

```go
import "github.com/heroku/x/cmdutil/v2/service"

var cfg config
svc := service.New(&cfg)
svc.Add(/* servers */)
svc.Run()
```

See [MIGRATION.md](MIGRATION.md) for migration details.
See [benchmarks/](benchmarks/) for performance comparisons.
