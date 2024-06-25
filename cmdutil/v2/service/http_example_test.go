package service_test

import (
	"io"
	"net/http"
	"time"

	"github.com/heroku/x/cmdutil/service"
)

func ExampleWithHTTPServerHook() {
	var cfg struct {
		Hello string `env:"HELLO,default=hello"`
	}
	svc := service.New(&cfg)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, cfg.Hello)
	})

	configureHTTP := func(s *http.Server) {
		s.ReadTimeout = 10 * time.Second
	}

	svc.Add(service.HTTP(svc.Logger, svc.MetricsProvider, handler,
		service.WithHTTPServerHook(configureHTTP),
	))

	svc.Run()
}
