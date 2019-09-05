package service

import (
	"bufio"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"testing"

	"github.com/heroku/x/testing/testlog"
)

func TestStandardHTTPServer(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, "OK")
		}),
		Addr: "127.0.0.1:0",
	}

	listenHook = make(chan net.Listener)
	defer func() { listenHook = nil }()

	s := standardServer(l, srv)

	done := make(chan struct{})
	go func() {
		s.Run()
		close(done)
	}()

	addr := (<-listenHook).Addr().String()

	res, err := http.Get("http://" + addr)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	data, _ := ioutil.ReadAll(res.Body)
	if string(data) != "OK" {
		t.Fatalf("want OK got %v", string(data))
	}

	s.Stop(nil)

	<-done
}

func TestBypassHTTPServer(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, "OK")
		}),
		Addr: "127.0.0.1:0",
	}

	listenHook = make(chan net.Listener)
	defer func() { listenHook = nil }()

	s := bypassServer(l, srv)

	done := make(chan struct{})
	go func() {
		s.Run()
		close(done)
	}()

	addr := (<-listenHook).Addr().String()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	io.WriteString(conn, "PROXY TCP4 127.0.0.1 127.0.0.1 44444 55555\n")

	req, _ := http.NewRequest("GET", "http://"+addr, nil)
	req.Write(conn)

	r := bufio.NewReader(conn)
	res, err := http.ReadResponse(r, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	data, _ := ioutil.ReadAll(res.Body)
	if string(data) != "OK" {
		t.Fatalf("want OK got %v", string(data))
	}

	s.Stop(nil)

	<-done
}
