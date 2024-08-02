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
	logger      logrus.FieldLogger
	addr        string
	done        chan struct{}
	pprofServer *http.Server
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

	if s.pprofServer != nil {
		go func() {
			if err := s.pprofServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				s.logger.WithError(err).Error("pprof server error")
			}
		}()
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

	if s.pprofServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.pprofServer.Shutdown(ctx); err != nil {
			s.logger.WithError(err).Error("Error shutting down pprof server")
		}
	}
}

// ProfileConfig holds the configuration for the pprof server.
type ProfileConfig struct {
	Addr            string
	ProfileHandlers map[string]http.HandlerFunc
}

// NewPprofServer sets up a pprof server with configurable profiling types and returns a Server instance.
func NewPprofServer(config ProfileConfig, l logrus.FieldLogger) *Server {
	if config.Addr == "" {
		config.Addr = "127.0.0.1:9998" // Default port
	}

	// Create a new HTTP mux for handling pprof routes.
	mux := http.NewServeMux()

	// Iterate over the profile handlers and add them to the mux.
	for profile, handler := range config.ProfileHandlers {
		if handler != nil {
			if profile == "mutex" {
				runtime.SetMutexProfileFraction(2)
			}
			mux.HandleFunc("/debug/pprof/"+profile, handler)
			l.WithFields(logrus.Fields{
				"at":      "adding",
				"service": "pprof",
				"profile": profile,
			}).Info("Added pprof profile handler")
		} else {
			l.WithFields(logrus.Fields{
				"at":      "ignoring",
				"service": "pprof",
				"profile": profile,
			}).Warn("Unknown pprof profile type")
		}
	}

	l.WithFields(logrus.Fields{
		"at":      "binding",
		"service": "pprof",
		"addr":    config.Addr,
	}).Info()

	// Create a new HTTP server for serving pprof endpoints.
	httpServer := &http.Server{
		Addr:              config.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &Server{
		logger:      l,
		addr:        config.Addr,
		done:        make(chan struct{}),
		pprofServer: httpServer,
	}
}
