package grpcclient

import (
	"sync"

	"google.golang.org/grpc"
)

var (
	cconnsMu sync.RWMutex
	cconns   = map[string]*grpc.ClientConn{}
)

// Conn returns the global grpc.ClientConn for the given service. If the
// connection has not yet been initialized, it will panic.
func Conn(service string) *grpc.ClientConn {
	cconnsMu.RLock()
	defer cconnsMu.RUnlock()
	c, ok := cconns[service]
	if !ok {
		panic("gRPC client connection not initialized")
	}
	return c
}

// RegisterConnection registers the given gRPC connection for usage under the
// specified service name.
func RegisterConnection(service string, cconn *grpc.ClientConn) {
	cconnsMu.Lock()
	defer cconnsMu.Unlock()
	cconns[service] = cconn
}

// DeregisterConnection deregisters the gRPC connection for the specified
// service name. The server is not stopped before it's deregistered. If no
// connection is registered for the service name, it will panic.
func DeregisterConnection(service string) {
	cconnsMu.Lock()
	defer cconnsMu.Unlock()
	if _, ok := cconns[service]; !ok {
		panic("gRPC client connection not initialized")
	}
	delete(cconns, service)
}
