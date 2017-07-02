package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
)

func ExampleMerge() {
	fmt.Println(merge(map[string]string{"A": "1"}, []string{"B=2"}))
	// Output: [B=2 A=1]
}

func ExampleInput() {
	tmpfile, err := ioutil.TempFile("", "testing")
	if err != nil {
		log.Fatal(err)
	}
	tmpfile.Write([]byte(`{"A": "1"}`))
	os.Stdin = tmpfile
	r, err := input()
	if err != nil {
		log.Fatal(err)
	}
	// _ = r
	// io.Copy(os.Stdout, tmpfile)
	io.Copy(os.Stdout, r)
	// Output: A: 1
}
