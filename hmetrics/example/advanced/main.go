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

	hml := log.New(os.Stderr, "heroku metrics", 0)
	if err := hmetrics.Report(ctx, hml); err != nil {
		if f, ok := err.(fataler); ok {
			if f.Fatal() {
				log.Fatal(err)
			}
			log.Println(err)
		}
	}

	http.ListenAndServe(":"+os.Getenv("PORT"), nil)
}
