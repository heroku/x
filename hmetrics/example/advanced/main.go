package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/heroku/x/hmetrics"
)

type fataler interface {
	Fatal() bool
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eh := func(err error) error {
		log.Println("Error reporting metrics to heroku:", err)
		return nil
	}

	go func() {
		for {
			if err := hmetrics.Report(ctx, eh); err != nil {
				if f, ok := err.(fataler); ok && f.Fatal() {
					log.Fatal(err)
				}
				log.Println(err)
			}
		}
	}()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	http.ListenAndServe(":"+port, nil)
}
