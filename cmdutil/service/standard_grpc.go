package service

import (
	"crypto/tls"

	"github.com/joeshaw/envdecode"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/heroku/x/cmdutil"
	"github.com/heroku/x/cmdutil/health"
	"github.com/heroku/x/cmdutil/spaceca"
	"github.com/heroku/x/go-kit/metrics"
	"github.com/heroku/x/grpc/grpcserver"
)

type grpcConfig struct {
	Bypass  bypassConfig
	SpaceCA spaceCAConfig
}

func loadMutualTLSCert(cfg grpcConfig) (tls.Certificate, [][]byte, error) {
	if cfg.SpaceCA.UseSpaceCA {
		ca := spaceca.CA{
			RootCert: []byte(cfg.SpaceCA.RootCACert),
			Cert:     []byte(cfg.SpaceCA.SpaceCACert),
			Key:      []byte(cfg.SpaceCA.SpaceCAKey),
		}
		domain := cfg.SpaceCA.Domain

		cert, err := spaceca.NewCACertificate(domain, ca)
		if err != nil {
			return tls.Certificate{}, nil, errors.Wrap(err, "error generating cert from spaceCA")
		}

		serverCACertList := [][]byte{ca.RootCert}

		if cfg.SpaceCA.RootCACertAlternate != "" {
			serverCACertList = append(serverCACertList, []byte(cfg.SpaceCA.RootCACertAlternate))
		}

		return *cert, serverCACertList, nil
	}

	serverCert, err := tls.X509KeyPair([]byte(cfg.Bypass.TLS.ServerCert), []byte(cfg.Bypass.TLS.ServerKey))
	if err != nil {
		return tls.Certificate{}, nil, errors.Wrap(err, "creating X509 key pair")
	}
	serverCACertList := [][]byte{[]byte(cfg.Bypass.TLS.ServerCACert)}

	return serverCert, serverCACertList, nil
}

// GRPC returns a standard GRPC server for the provided handler.
// Router-bypass and TLS config are inferred from the environment.
//
// Currently only supports running in router-bypass mode, unlike HTTP.
func GRPC(
	l logrus.FieldLogger,
	m metrics.Provider,
	server grpcserver.Starter,
	grpcOpts ...grpcserver.ServerOption) cmdutil.Server {
	var cfg grpcConfig
	envdecode.MustDecode(&cfg)

	cert, serverCACertList, err := loadMutualTLSCert(cfg)
	if err != nil {
		l.WithError(err).Fatal()
	}

	var srvs []cmdutil.Server

	if cfg.Bypass.SecurePort != 0 {
		grpcOpts = append(grpcOpts, grpcserver.MetricsProvider(m))

		srvs = append(srvs, grpcserver.NewStandardServer(
			l,
			cfg.Bypass.SecurePort,
			serverCACertList,
			cert,
			server,
			grpcOpts...,
		))
	}

	if cfg.Bypass.HealthPort != 0 {
		srvs = append(srvs, health.NewTCPServer(l, m, health.Config{
			Port: cfg.Bypass.HealthPort,
		}))
	}

	return cmdutil.MultiServer(srvs...)
}
