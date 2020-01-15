package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/golang/protobuf/proto"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"

	"github.com/heroku/x/cmd/protoc-gen-loggingtags/internal/gen"
)

var file = flag.String("file", "stdin", "where to load data from")

func main() {
	flag.Parse()
	f := os.Stdin
	if *file != "stdin" {
		f, _ = os.Open("input.txt")
	}
	input, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatal(err)
	}
	req := new(plugin.CodeGeneratorRequest)
	if err = proto.Unmarshal(input, req); err != nil {
		log.Fatal(err)
	}
	pkgMap := make(map[string]string)
	if param := req.GetParameter(); param != "" {
		for _, p := range strings.Split(param, ",") {
			spec := strings.SplitN(p, "=", 2)
			name, value := spec[0], spec[1]
			if strings.HasPrefix(name, "M") {
				pkgMap[name[1:]] = value
				continue
			}
		}
	}

	resp, err := gen.Generate(req, pkgMap)
	if err != nil {
		log.Fatal(err)
	}
	buf, err := proto.Marshal(resp)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := os.Stdout.Write(buf); err != nil {
		log.Fatal(err)
	}
}
