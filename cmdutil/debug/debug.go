// Package debug wraps the gops agent for use as a cmdutil-compatible Server.
//
// The debug server will be started on DEBUG_PORT (default 9999). Get a stack
// trace, profile memory, etc. by running the gops command line connected to
// locahost:9999 like:
//
//		$ gops stack localhost:9999
//		goroutine 50 [running]:
//		  runtime/pprof.writeGoroutineStacks(0x4a18a20, 0xc000010138, 0x0, 0x0)
//		  	/usr/local/Cellar/go/1.13.5/libexec/src/runtime/pprof/pprof.go:679 +0x9d
//		  runtime/pprof.writeGoroutine(0x4a18a20, 0xc000010138, 0x2, 0x0, 0x0)
//		  ...
//
// Learn more about gops at https://github.com/google/gops.
package debug

import (
	"fmt"

	"github.com/google/gops/agent"
	"github.com/sirupsen/logrus"
)

// New inializes a debug server listening on the provided port.
//
// Connect to the debug server with gops:
//		gops stack localhost:PORT
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
