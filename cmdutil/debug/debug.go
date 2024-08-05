// Package debug wraps the gops agent for use as a cmdutil-compatible Server.
//
// The debug server will be started on DEBUG_PORT (default 9999). Get a stack
// trace, profile memory, etc. by running the gops command line connected to
// locahost:9999 like:
//
//	$ gops stack localhost:9999
//	goroutine 50 [running]:
//	  runtime/pprof.writeGoroutineStacks(0x4a18a20, 0xc000010138, 0x0, 0x0)
//	  	/usr/local/Cellar/go/1.13.5/libexec/src/runtime/pprof/pprof.go:679 +0x9d
//	  runtime/pprof.writeGoroutine(0x4a18a20, 0xc000010138, 0x2, 0x0, 0x0)
//	  ...
//
// Learn more about gops at https://github.com/google/gops.
package debug

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"runtime"
	"time"

	"github.com/google/gops/agent"
	"github.com/sirupsen/logrus"
)

// New inializes a debug server listening on the provided port.
//
// Connect to the debug server with gops:
//
//	gops stack localhost:PORT
func New(l logrus.FieldLogger, port int) *Server {
	return &Server{
		logger: l,
		addr:   fmt.Sprintf("127.0.0.1:%d", port),
		done:   make(chan struct{}),
	}
}

// Server wraps a gops server for easy use with oklog/group.
type Server struct {
	logger logrus.FieldLogger
	addr   string
	done   chan struct{}
}

// Run starts the debug server.
//
// It implements oklog group's runFn.
func (s *Server) Run() error {
	s.logger.WithFields(logrus.Fields{
		"at":      "binding",
		"service": "debug",
		"addr":    s.addr,
	}).Info()

	opts := agent.Options{
		Addr:            s.addr,
		ShutdownCleanup: false,
	}
	if err := agent.Listen(opts); err != nil {
		return err
	}

	<-s.done
	return nil
}

// Stop shuts down the debug server.
//
// It implements oklog group's interruptFn.
func (s *Server) Stop(_ error) {
	agent.Close()

	close(s.done)
}

// PProfServer wraps a pprof server.
type PProfServer struct {
	logger        logrus.FieldLogger
	addr          string
	done          chan struct{}
	pprofServer   *http.Server
	profileConfig PProfServerConfig
}

// ProfileConfig holds the configuration for the pprof server.
type PProfServerConfig struct {
	Addr                 string
	ProfileNames         []string
	MutexProfileFraction int
}

// NewPProfServer sets up a pprof server with configurable profiling types and returns a PProfServer instance.
func NewPProfServer(config PProfServerConfig, l logrus.FieldLogger) *PProfServer {
	if config.Addr == "" {
		config.Addr = "127.0.0.1:9998" // Default port
	}

	// Create a new HTTP mux for handling pprof routes.
	mux := http.NewServeMux()

	// Iterate over the handlers and add them to the mux.
	for _, profile := range config.ProfileNames {
		if profile == "mutex" {
			if config.MutexProfileFraction == 0 {
				config.MutexProfileFraction = 2 // Use default value of 2 if not set
			}
			runtime.SetMutexProfileFraction(config.MutexProfileFraction)
		}
		mux.Handle("/debug/pprof/"+profile, pprof.Handler(profile))
	}

	httpServer := &http.Server{
		Addr:              config.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &PProfServer{
		logger:        l,
		addr:          config.Addr,
		done:          make(chan struct{}),
		pprofServer:   httpServer,
		profileConfig: config,
	}
}

// Run starts the pprof server.
//
// It implements oklog group's runFn.
func (s *PProfServer) Run() error {
	s.logger.WithFields(logrus.Fields{
		"at":      "binding",
		"service": "pprof",
		"addr":    s.addr,
	}).Info()

	if s.pprofServer != nil {
		if err := s.pprofServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return err
		}
	}

	<-s.done
	return nil
}

// Stop shuts down the pprof server.
//
// It implements oklog group's interruptFn.
func (s *PProfServer) Stop(_ error) {
	if s.pprofServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.pprofServer.Shutdown(ctx); err != nil {
			s.logger.WithError(err).Error("Error shutting down pprof server")
		}
	}
	close(s.done)
}
