package healthcheck

import (
	"net"
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/rs/zerolog"

	hmetrics "github.com/heroku/x/go-kit/metrics"
)

// TCPServer answers healthcheck requests from TCP routers, such as an ELB.
type TCPServer struct {
	logger  zerolog.Logger
	addr    string
	ln      net.Listener
	counter metrics.Counter
}

// NewTCPServer initializes a new health-check server.
func NewTCPServer(logger zerolog.Logger, provider hmetrics.Provider, addr string) *TCPServer {
	return &TCPServer{
		logger:  logger,
		counter: provider.NewCounter("health"),
		addr:    addr,
	}
}

// Run listens on the configured address and responds to healthcheck requests
// from TCP routers, such as an ELB.
func (s *TCPServer) Run() error {
	if err := s.start(); err != nil {
		return err
	}

	return s.serve()
}

// Stop shuts down the TCPServer if it was already started.
//
// Stop implements the kit.Server interface.
func (s *TCPServer) Stop(error) {
	if s.ln != nil {
		s.ln.Close()
	}
}

func (s *TCPServer) start() error {
	s.logger.Info().
		Str("at", "bind").
		Str("addr", s.addr).
		Send()

	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}

	s.ln = ln
	return nil
}

func (s *TCPServer) serve() error {
	const retryDelay = 50 * time.Millisecond

	for {
		conn, err := s.ln.Accept()
		if err != nil {
			if e, ok := err.(net.Error); ok && e.Timeout() {
				s.logger.Error().
					Err(err).
					Str("at", "accept").
					Msgf("retrying in %s", retryDelay)

				time.Sleep(retryDelay)
				continue
			}

			return err
		}

		go s.serveConn(conn)
	}
}

func (s *TCPServer) serveConn(conn net.Conn) {
	defer conn.Close()

	s.counter.Add(1)

	if _, err := conn.Write([]byte("OK\n")); err != nil {
		s.logger.Error().Err(err).Send()
	}
}
