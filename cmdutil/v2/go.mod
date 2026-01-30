module github.com/heroku/x/cmdutil/v2

go 1.25.1

require (
	github.com/gomodule/redigo v1.8.9
	github.com/google/gops v0.3.22
	github.com/heroku/x v0.5.2
	github.com/joeshaw/envdecode v0.0.0-20200121155833-099f1fc765bd
	github.com/oklog/run v1.1.0
	github.com/pkg/errors v0.9.1
	github.com/urfave/cli v1.21.0
	go.opentelemetry.io/contrib/instrumentation/runtime v0.45.0
	go.opentelemetry.io/otel v1.38.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.27.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.27.0
	go.opentelemetry.io/otel/sdk v1.36.0
	go.opentelemetry.io/otel/sdk/metric v1.36.0
	google.golang.org/grpc v1.64.0
)

require (
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/pprof v0.0.0-20260115054156-294ebfa9ad83
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.20.0 // indirect
	github.com/heroku/x/testing/testlog/v2 v2.0.0
	github.com/lstoll/grpce v1.7.0 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel/metric v1.38.0
	go.opentelemetry.io/otel/trace v1.38.0 // indirect
	go.opentelemetry.io/proto/otlp v1.2.0
	golang.org/x/net v0.33.0 // indirect
	golang.org/x/sync v0.11.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.22.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240520151616-dc85e6b867a5 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240515191416-fc5f0ca64291 // indirect
	google.golang.org/protobuf v1.34.1 // indirect
)

replace github.com/heroku/x/testing/testlog/v2 => ../../testing/testlog/v2
