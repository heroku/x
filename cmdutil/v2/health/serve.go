// Package health provides cmdutil-compatible healthcheck utilities.
package health

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"go.opentelemetry.io/otel/metric"

	"github.com/heroku/x/cmdutil"
	"github.com/heroku/x/tickgroup"
)

// NewTCPServer returns a cmdutil.Server which emits a health metric whenever a TCP
// connection is opened on the configured port.
func NewTCPServer(logger *slog.Logger, meter metric.Meter, cfg Config) cmdutil.Server {
	logger.Info("binding", slog.String("service", "healthcheck"), slog.Int("port", cfg.Port))

	counter, _ := meter.Int64Counter("health")

	return &tcpServer{
		logger:  logger,
		counter: counter,
		addr:    fmt.Sprintf(":%d", cfg.Port),
	}
}

type tcpServer struct {
	logger  *slog.Logger
	addr    string
	ln      net.Listener
	counter metric.Int64Counter
}

func (s *tcpServer) Run() error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	s.ln = ln

	const retryDelay = 50 * time.Millisecond
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			if e, ok := err.(net.Error); ok && e.Timeout() {
				s.logger.Error("accept error, retrying", slog.Any("error", err), slog.Duration("retry", retryDelay))
				time.Sleep(retryDelay)
				continue
			}
			return err
		}
		go s.serveConn(conn)
	}
}

func (s *tcpServer) Stop(error) {
	if s.ln != nil {
		s.ln.Close()
	}
}

func (s *tcpServer) serveConn(conn net.Conn) {
	defer conn.Close()
	s.counter.Add(context.Background(), 1)
	if _, err := conn.Write([]byte("OK\n")); err != nil {
		s.logger.Error("write error", slog.Any("error", err))
	}
}

// NewTickingServer returns a cmdutil.Server which emits a health metric every
// cfg.MetricInterval seconds.
func NewTickingServer(logger *slog.Logger, meter metric.Meter, cfg Config) cmdutil.Server {
	logger.Info("starting",
		slog.String("service", "healthcheck-worker"),
		slog.Int("interval", cfg.MetricInterval))

	counter, _ := meter.Int64Counter("health")

	return cmdutil.NewContextServer(func(ctx context.Context) error {
		g := tickgroup.New(ctx)
		g.Go(time.Duration(cfg.MetricInterval)*time.Second, func() error {
			counter.Add(ctx, 1)
			return nil
		})
		return g.Wait()
	})
}
