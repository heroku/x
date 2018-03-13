package scrub

import (
	"log"
	"net/http"
	"net/url"
)

func ExampleHeader() {
	h := http.Header{
		"Authorization": []string{"Basic hunter2"},
	}

	scrubbed := Header(h)
	val := scrubbed.Get("Authorization") // Will be `Basic [SCRUBBED]`
	_ = val                              // do something with `val`
}

func ExampleURL() {
	u, err := url.Parse("https://google.com?api_key=hunter2")
	if err != nil {
		log.Fatal(err)
	}

	su := URL(u)
	log.Println(su.String()) // should be `https://google.com?api_key=[SCRUBBED]`
}
