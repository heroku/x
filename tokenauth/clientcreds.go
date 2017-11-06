package tokenauth

import "context"

// RPCCredentials are a grpc.PerRPCCredentials that will be passed in RPC requests
// to authorize the call.
type RPCCredentials struct {
	// Credentials to be passed to the remote authorizer(s). Note - all keys
	// should be lowercase, the grpc transport will downcase them regardless.
	Credentials map[string]string
	// AllowInsecure determines if these are acceptable on connections deemed
	// insecure.
	AllowInsecure bool
}

// GetRequestMetadata gets the request metadata as a map from a Credentials. These will be prefixed
// appropriately in the metadata
func (r *RPCCredentials) GetRequestMetadata(ctx context.Context, _ ...string) (map[string]string, error) {
	md := map[string]string{}
	for k, v := range r.Credentials {
		md[metadataCredsPrefix+k] = v
	}
	return md, nil
}

// RequireTransportSecurity indicates whether the credentials requires transport security.
func (r *RPCCredentials) RequireTransportSecurity() bool {
	return !r.AllowInsecure
}
