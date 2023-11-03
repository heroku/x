package otel

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"

	xmetrics "github.com/heroku/x/go-kit/metrics"
)

const (
	defaultExponentialHistogramMaxSize  = 160
	defaultExponentialHistogramMaxScale = 20
)

var DefaultReaderInterval = time.Minute

type config struct {
	ctx                 context.Context // used for init and shutdown of the otlp exporter and other bits of this Provider
	serviceResource     *resource.Resource
	prefix              string
	collectPeriod       time.Duration
	temporalitySelector metric.TemporalitySelector
	aggregationSelector metric.AggregationSelector
	exporterFactory     exporterFactory
}

// Provider initializes a global otlp meter provider that can collect metrics and
// use a collector to push those metrics to various backends (e.g. Argus, Honeycomb).
// Initialize with New(...). An initialized Provider must be started before
// it can be used to provide a Meter (i.e. p.Start()).
type Provider struct {
	cfg           *config
	meterProvider *metric.MeterProvider
	viewCache     *viewCache

	mu         sync.Mutex
	counters   map[string]*Counter
	gauges     map[string]*Gauge
	histograms map[string]*Histogram
}

type viewCache struct {
	lock    sync.RWMutex
	streams map[string]metric.Stream
}

func (v *viewCache) Store(stream metric.Stream) {
	v.lock.Lock()
	defer v.lock.Unlock()

	v.streams[stream.Name] = stream
}

func (v *viewCache) View(i metric.Instrument) (metric.Stream, bool) {
	v.lock.RLock()
	defer v.lock.RUnlock()

	stream, ok := v.streams[i.Name]

	// copy these items over, the `name` should already be consistent
	stream.Unit = i.Unit
	stream.Description = i.Description

	return stream, ok
}

type exporterFactory func(*config) (metric.Exporter, error)

// New returns a new, unstarted Provider. Use its Start() method to start
// and establish a connection with its exporter's collector agent.
func New(ctx context.Context, serviceName string, opts ...Option) (xmetrics.Provider, error) {
	cfg := &config{
		ctx:             ctx,
		collectPeriod:   DefaultReaderInterval,
		serviceResource: resource.Default(), // this fetches from env by default and pre-populates some fields.
	}
	defaultOpts := []Option{
		WithServiceStandard(serviceName),
		DefaultAggregationSelector(),
		DefaultEndpointExporter(),
	}

	opts = append(defaultOpts, opts...)
	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, fmt.Errorf("failed to apply options: %w", err)
		}
	}

	p := Provider{
		cfg:        cfg,
		counters:   make(map[string]*Counter),
		gauges:     make(map[string]*Gauge),
		histograms: make(map[string]*Histogram),
		viewCache: &viewCache{
			streams: make(map[string]metric.Stream),
		},
	}

	exporter, err := cfg.exporterFactory(cfg)
	if err != nil {
		return nil, err
	}

	reader := metric.NewPeriodicReader(
		exporter,
		metric.WithInterval(cfg.collectPeriod),
	)

	meterProvider := metric.NewMeterProvider(
		metric.WithResource(cfg.serviceResource),
		metric.WithReader(reader),
		metric.WithView(p.viewCache.View),
	)

	// initialize the metricProvider
	p.meterProvider = meterProvider

	return &p, nil
}

// Start starts the provider's controller and exporter.
func (p *Provider) Start() error {
	return nil
}

// Stop shuts down the provider's controller and exporter.
// It should be used to ensure the metrics provider drains all metrics before exiting the program.
func (p *Provider) Stop() {
	_ = p.meterProvider.Shutdown(p.cfg.ctx)
}

// Flush stops and starts the controller in order to flush metrics
// immediately, without having to wait until the next collection occurs.
// The flush is synchronous and returns an error if the controller fails to
// flush or cannot restart after flushing.
func (p *Provider) Flush() error {
	return p.meterProvider.ForceFlush(p.cfg.ctx)
}

var ExplicitAggregationSelector = metric.DefaultAggregationSelector

func ExponentialAggregationSelector(ik metric.InstrumentKind) metric.Aggregation {
	switch ik {
	case
		metric.InstrumentKindCounter,
		metric.InstrumentKindUpDownCounter,
		metric.InstrumentKindObservableCounter,
		metric.InstrumentKindObservableUpDownCounter:
		return metric.AggregationSum{}
	case metric.InstrumentKindObservableGauge:
		return metric.AggregationLastValue{}
	case metric.InstrumentKindHistogram:
		return metric.AggregationBase2ExponentialHistogram{
			MaxSize:  defaultExponentialHistogramMaxSize,
			MaxScale: defaultExponentialHistogramMaxScale,
			NoMinMax: false,
		}
	}
	panic("unknown instrument kind")
}
