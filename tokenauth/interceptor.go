package tokenauth

import (
	"strings"

	"google.golang.org/grpc/codes"

	"github.com/grpc-ecosystem/go-grpc-middleware"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// UnaryServerInterceptor returns a grpc.UnaryServerInterceptor that enables the
// authentication and authorization of gRPC calls using the Authorizer
// interface.
func UnaryServerInterceptor(authorizer Authorizer) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, in interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		newCtx, err := authCall(ctx, authorizer, info.FullMethod)
		if err != nil {
			return nil, err
		}
		return handler(newCtx, in)
	}
}

// StreamServerInterceptor returns a grpc.StreamServerInterceptor that enables
// the authentication and authorization of gRPC calls using the Authorizer
// interface.
func StreamServerInterceptor(authorizer Authorizer) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		newCtx, err := authCall(stream.Context(), authorizer, info.FullMethod)
		if err != nil {
			return err
		}
		ws := grpc_middleware.WrapServerStream(stream)
		ws.WrappedContext = newCtx
		return handler(srv, ws)
	}
}

// authCall runs the auth on the passed call information.
func authCall(ctx context.Context, authorizer Authorizer, method string) (context.Context, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, grpc.Errorf(codes.Unauthenticated, "Context does not contain any metadata")
	}
	c := map[string]string{}
	for k, v := range md {
		if strings.HasPrefix(k, metadataCredsPrefix) {
			c[strings.TrimPrefix(k, metadataCredsPrefix)] = v[0]
		}
	}

	// method looks like /package.service/method
	p := strings.Split(method, "/")
	if len(p) != 3 || p[0] != "" || p[1] == "" || p[2] == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "invalid RPC request")
	}
	pp := strings.Split(p[1], ".")
	if len(pp) != 2 || pp[0] == "" || pp[1] == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "invalid RPC request")
	}
	call := &RPC{Package: pp[0], Service: pp[1], Method: p[2]}

	return authorizer.Authorize(ctx, call, c)
}
