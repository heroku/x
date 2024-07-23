package service

import (
	"testing"

	"github.com/joeshaw/envdecode"
)

// httpConfig should be decodable with nothing required.
//
// This isn't a perfect test as there may be something set
// in the test environment that is used by httpConfig but it
// should help ensure at least more specific items like
// SERVER_CA_CERT are not required.
func TestDecodeHTTPConfig(t *testing.T) {
	var cfg httpConfig

	if err := envdecode.StrictDecode(&cfg); err != nil {
		t.Fatal(err)
	}
}

// grpcConfig should be decodable with nothing required.
//
// This isn't a perfect test as there may be something set
// in the test environment that is used by grpcConfig but it
// should help ensure at least more specific items like
// SERVER_CA_CERT are not required.
func TestDecodeGRPCConfig(t *testing.T) {
	var cfg grpcConfig

	if err := envdecode.StrictDecode(&cfg); err != nil {
		t.Fatal(err)
	}
}
