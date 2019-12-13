package h2c

import (
	"bufio"
	"net"
	"net/http"
	"strings"

	"golang.org/x/net/http2"
)

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

	new(http2.Server).ServeConn(bufConn{conn, bufrw}, &http2.ServeConnOpts{
		Handler: s.HTTP2Handler,
	})
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
