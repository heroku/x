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
	"errors"
	"fmt"
	"net/http"
	"net/http/pprof"
	"runtime"
	"sync"
	"time"

	"github.com/google/gops/agent"
	"github.com/sirupsen/logrus"
)

// New inializes a debug server listening on the provided port.
//
// Connect to the debug server with gops:
//
//	gops stack localhost:PORT
func New(l logrus.FieldLogger, config Config) *Server {
	server := &Server{
		logger: l,
		addr:   fmt.Sprintf("127.0.0.1:%d", config.Port),
		done:   make(chan struct{}),
	}
	if config.Enabled {
		server.pprof = NewPProfServer(l, &config.PProf)
	}
	return server
}

// Server wraps a gops server for easy use with oklog/group.
type Server struct {
	logger logrus.FieldLogger
	addr   string
	done   chan struct{}
	pprof  *PProfServer
}

// Run starts the debug server.
//
// It implements oklog group's runFn and pprof.
func (s *Server) Run() error {

	var wg sync.WaitGroup
	gopsErrChan := make(chan error, 1)
	pprofErrChan := make(chan error, 1)

	wg.Add(1)
	go func() {
		defer wg.Done()
		gopsErrChan <- s.RunGOPS()
	}()

	if s.pprof != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pprofErrChan <- s.pprof.Run()
		}()
	}

	wg.Wait()
	gopsErr := <-gopsErrChan
	var pprofErr error
	if s.pprof != nil {
		pprofErr = <-pprofErrChan
	}

	var err error
	if gopsErr != nil {
		err = fmt.Errorf("gops error: %w", gopsErr)
	}
	if pprofErr != nil {
		errPProf := fmt.Errorf("pprof error: %w", pprofErr)
		err = errors.Join(err, errPProf)
	}

	return err
}

// Run starts the debug server.
//
// It implements oklog group's runFn.
func (s *Server) RunGOPS() error {
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
// It implements oklog group's interruptFn and pprof stop.
func (s *Server) Stop(_ error) {
	agent.Close()

	close(s.done)

	if s.pprof != nil {
		s.pprof.Stop(nil)
	}
}

// PProfServer wraps a pprof server.
type PProfServer struct {
	logger      logrus.FieldLogger
	addr        string
	done        chan struct{}
	pprofServer *http.Server
}

// NewPProfServer sets up a pprof server with configurable profiling types and returns a PProfServer instance.
func NewPProfServer(l logrus.FieldLogger, pprofConfig *PProf) *PProfServer {

	runtime.MemProfileRate = pprofConfig.MemProfileRate

	if pprofConfig.EnableMutexProfiling {
		runtime.SetMutexProfileFraction(pprofConfig.MutexProfileFraction)
	}

	if pprofConfig.EnableBlockProfiling {
		runtime.SetBlockProfileRate(pprofConfig.BlockProfileRate)
	}

	httpServer := &http.Server{
		Addr:              fmt.Sprintf("127.0.0.1:%d", pprofConfig.Port),
		Handler:           http.HandlerFunc(pprof.Index),
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &PProfServer{
		logger:      l,
		addr:        httpServer.Addr,
		done:        make(chan struct{}),
		pprofServer: httpServer,
	}
}

// Run starts the pprof server.
//
// It implements oklog group's runFn.
func (s *PProfServer) Run() error {
	if s.pprofServer == nil {
		return fmt.Errorf("pprofServer is nil")
	}

	s.logger.WithFields(logrus.Fields{
		"at":      "binding",
		"service": "pprof",
		"addr":    s.addr,
	}).Info()

	if err := s.pprofServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
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
