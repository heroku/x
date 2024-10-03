package h2c

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/http2"
)

const shutdownPollInterval = 500 * time.Millisecond

// Server is an HTTP 1.1 server that can detect h2c upgrades and serve them by
// an HTTP2 handler.
type Server struct {
	HTTP2Handler      http.Handler
	NonUpgradeHandler http.Handler
	// ALBSupport can be used to enable this listener to work behind a AWS ALB.
	// These strip the Connection header for non-websocket upgrades, so we only
	// use the Upgrade header in this case. This is not to spec, but seems to
	// work OK.
	ALBSupport bool

	connections   map[net.Conn]struct{}
	connectionsMu sync.Mutex
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	connection, upgrade := r.Header.Get("Connection"), r.Header.Get("Upgrade")

	if !s.isH2C(connection, upgrade) {
		s.NonUpgradeHandler.ServeHTTP(w, r)
		return
	}

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Connection", "Upgrade")
	w.Header().Set("Upgrade", "h2c")
	w.WriteHeader(http.StatusSwitchingProtocols)

	conn, bufrw, err := hj.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.trackConn(conn, true)
	defer s.trackConn(conn, false)

	new(http2.Server).ServeConn(bufConn{conn, bufrw}, &http2.ServeConnOpts{
		Handler: s.HTTP2Handler,
	})
}

// Shutdown blocks until all connections have completed.
// Note that this will not actually close the listener, since we don't have
// access to it here. Instead, it is assumed that the caller has already
// closed the underlying listener prior to calling Shutdown.
func (s *Server) Shutdown(ctx context.Context) error {
	ticker := time.NewTicker(shutdownPollInterval)
	defer ticker.Stop()
	for {
		if s.noActiveConnections() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *Server) trackConn(conn net.Conn, add bool) {
	s.connectionsMu.Lock()
	defer s.connectionsMu.Unlock()

	if s.connections == nil {
		s.connections = make(map[net.Conn]struct{})
	}
	if add {
		s.connections[conn] = struct{}{}
	} else {
		delete(s.connections, conn)
	}
}

func (s *Server) noActiveConnections() bool {
	s.connectionsMu.Lock()
	defer s.connectionsMu.Unlock()

	// This is a pretty naive approach.
	// Just checking whether there are any connections used for requests
	// that haven't yet finished. However, it's "good enough".
	return len(s.connections) == 0
}

func (s *Server) isH2C(connection, upgrade string) bool {
	connection, upgrade = strings.ToLower(connection), strings.ToLower(upgrade)
	return upgrade == "h2c" && (s.ALBSupport || connection == "upgrade" || strings.HasPrefix(connection, "upgrade,"))
}

type bufConn struct {
	net.Conn
	bufrw *bufio.ReadWriter
}

func (bc bufConn) Close() error {
	bc.bufrw.Flush()

	return bc.Conn.Close()
}

func (bc bufConn) Read(p []byte) (int, error) {
	if n := bc.bufrw.Reader.Buffered(); n > 0 {
		return bc.bufrw.Read(p)
	}

	return bc.Conn.Read(p)
}

func (bc bufConn) Write(p []byte) (int, error) {
	if n := bc.bufrw.Writer.Buffered(); n > 0 {
		if err := bc.bufrw.Flush(); err != nil {
			return 0, err
		}
	}

	return bc.Conn.Write(p)
}
