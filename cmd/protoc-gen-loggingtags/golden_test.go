package main

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
)

func TestMain(m *testing.M) {
	if os.Getenv("RUN_AS_PLUGIN") != "" {
		main()
		os.Exit(0)
	}

	os.Exit(m.Run())
}

var protoTmpl = template.Must(template.New("proto").Parse(`
syntax = "proto3";
{{ if .ImportPath }}import "{{ .ImportPath }}";{{ end }}

package foo;

message T {
	string field = 1{{ if .Annotate }} [(heroku.loggingtags.safe) = true]{{ end }};
}
`))

func TestGenerate(t *testing.T) {
	type testCase struct { // nolint: maligned
		name       string
		ImportPath string
		Annotate   bool
		protocArgs []string

		wantGenerated bool
	}

	for _, tt := range []testCase{
		{
			name:          "not using loggingtags",
			wantGenerated: false,
		},
		{
			name:          "imported and used",
			ImportPath:    "loggingtags/safe.proto",
			Annotate:      true,
			wantGenerated: true,
		},
		{
			name:          "custom import path",
			ImportPath:    "safe.proto",
			Annotate:      true,
			protocArgs:    []string{"--proto_path=../../loggingtags"},
			wantGenerated: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			workdir, err := ioutil.TempDir("", "proto-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(workdir)

			f, err := os.Create(filepath.Join(workdir, "foo.proto"))
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()

			if err := protoTmpl.Execute(f, tt); err != nil {
				t.Fatal(err)
			}

			args := []string{
				"-I" + workdir,
				"--loggingtags_out=" + workdir,
			}
			args = append(args, tt.protocArgs...)
			args = append(args, filepath.Join(workdir, "foo.proto"))

			protoc(t, args)

			generated := "foo.pb.loggingtags.go"
			exists := fileExists(t, filepath.Join(workdir, generated))
			if exists != tt.wantGenerated {
				t.Errorf("stat(%v) = %v, want %v", generated, exists, tt.wantGenerated)
			}
		})
	}
}

func TestSample(t *testing.T) {
	workdir, err := ioutil.TempDir("", "proto-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(workdir)

	protoc(t, []string{
		"--proto_path=" + filepath.Join("internal", "test"),
		"--proto_path=.",
		"--loggingtags_out=" + workdir,
		filepath.Join("sample.proto"),
	})

	goldenPath := filepath.Join("internal", "test", "sample.pb.loggingtags.go")
	genPath := filepath.Join(workdir, "sample.pb.loggingtags.go")
	if !fileExists(t, genPath) {
		t.Fatal("want sample.pb.loggingtags.go to exist")
	}

	cmd := exec.Command("diff", "-u", goldenPath, genPath)
	out, _ := cmd.CombinedOutput()
	if len(out) > 0 {
		t.Errorf("golden file differs: %v\n%v", "sample.pb.loggingtags.go", string(out))
	}
}

func fileExists(t *testing.T, name string) bool {
	_, err := os.Stat(name)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}

		t.Error(err)
	}

	return true
}

func protoc(t *testing.T, args []string) {
	root, _ := filepath.Abs(filepath.Join("..", ".."))
	protoc := filepath.Join(root, ".tools", "bin", "protoc")

	if !fileExists(t, protoc) {
		cmd := exec.Command("make", protoc)
		cmd.Dir = "../.."
		out, err := cmd.CombinedOutput()
		if len(out) > 0 || err != nil {
			t.Log("RUNNING: ", strings.Join(cmd.Args, " "))
		}
		if len(out) > 0 {
			t.Log(string(out))
		}
		if err != nil {
			t.Fatalf("make: %v", err)
		}
	}

	cmd := exec.Command(protoc, "--plugin=protoc-gen-loggingtags="+os.Args[0], "--proto_path="+root)
	cmd.Args = append(cmd.Args, args...)
	// We set the RUN_AS_PLUGIN environment variable to indicate that
	// the subprocess should act as a proto compiler rather than a test.
	cmd.Env = append(os.Environ(), "RUN_AS_PLUGIN=1")
	out, err := cmd.CombinedOutput()
	if len(out) > 0 || err != nil {
		t.Log("RUNNING: ", strings.Join(cmd.Args, " "))
	}
	if len(out) > 0 {
		t.Log(string(out))
	}
	if err != nil {
		t.Fatalf("protoc: %v", err)
	}
}
