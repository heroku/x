package tokenauth

import "context"

const metadataCredsPrefix = "tokenauth-"

// RPC is a Remote Procedure Call made against a gRPC service.
type RPC struct {
	Package string `json:"package"`
	Service string `json:"service"`
	Method  string `json:"method"`
}

// Authorizer authorizes RPC requests. It is passed the details of the RPC
// call, and the credentials provided in the request. It will either return the
// context to use for the call, or an error if the credentials do not pass
type Authorizer interface {
	Authorize(ctx context.Context, rpc *RPC, credentials map[string]string) (newCtx context.Context, err error)
}
