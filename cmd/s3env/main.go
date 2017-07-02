package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"syscall"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/joeshaw/envdecode"
)

// config is used in production when you want to fetch a JSON config
// from an S3 object.
type config struct {
	Key    string `env:"S3ENV_KEY,default=env.json"`
	Bucket string `env:"S3ENV_BUCKET,required"`

	AccessID  string `env:"S3ENV_AWS_ACCESS_KEY_ID,required"`
	Region    string `env:"S3ENV_AWS_REGION,required"`
	SecretKey string `env:"S3ENV_AWS_SECRET_ACCESS_KEY,required"`
}

func main() {
	if len(os.Args) < 2 {
		abortf("Usage: %s <command>\n", os.Args[0])
	}

	cmd := os.Args[1]   // The wrapped command we'll execute.
	args := os.Args[1:] // syscall.Exec needs the args including cmd.

	exe, err := exec.LookPath(cmd)
	if err != nil {
		abortf("fatal: %s: %s\n", cmd, err)
	}

	// fetch all of the environment including the parent's env.
	env, err := environ()
	if err != nil {
		abortf("fatal: %s\n", err)
	}

	// pass control to the given cmd. This also means all signal
	// handling is delegated at this point to the cmd.
	if err = syscall.Exec(exe, args, env); err != nil {
		abortf("fatal: %s: %s\n", exe, err)
	}
}

func abortf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

// environ reads the input JSON (either via STDIN, or via the s3 object) and
// merges that with the parent env.
func environ() ([]string, error) {
	in, err := input()
	if err != nil {
		return nil, err
	}
	defer in.Close()

	env := make(map[string]string)
	if err = json.NewDecoder(in).Decode(&env); err != nil {
		return nil, err
	}
	return merge(env, os.Environ()), nil
}

// merge takes a map of envs and a slice and combines them into one slice.
// e.g. given "A" => 1, and []{"B=2"}, you get {"A=1", "B=2"}
func merge(env map[string]string, environ []string) []string {
	result := make([]string, 0, len(env)+len(environ))

	for _, kv := range environ {
		result = append(result, kv)
	}

	for k, v := range env {
		result = append(result, k+"="+v)
	}

	return result
}

// input gets the appropriate input source. If there was any data pumped into
// STDIN, we'll choose that. Otherwise we'll try to load the s3 object that
// was configured.
func input() (io.ReadCloser, error) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return nil, err
	}

	// If data is being pumped into STDIN, use that as our JSON input.
	// Useful for easy testing.
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		return ioutil.NopCloser(os.Stdin), nil
	}

	var cfg config
	if err := envdecode.StrictDecode(&cfg); err != nil {
		return nil, err
	}

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(cfg.Region),
		Credentials: credentials.NewStaticCredentials(
			cfg.AccessID,
			cfg.SecretKey,
			"",
		),
	})

	client := s3.New(sess)

	out, err := client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(cfg.Bucket),
		Key:    aws.String(cfg.Key),
	})
	if err != nil {
		return nil, err
	}
	return out.Body, nil
}
