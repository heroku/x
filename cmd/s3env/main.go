package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/joeshaw/envdecode"
)

type config struct {
	Key    string `env:"S3ENV_KEY,default=env.json"`
	Bucket string `env:"S3ENV_BUCKET,required"`

	AccessID  string `env:"S3ENV_AWS_ACCESS_KEY_ID,required"`
	Region    string `env:"S3ENV_AWS_REGION,required"`
	SecretKey string `env:"S3ENV_AWS_SECRET_ACCESS_KEY,required"`
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <command>\n", os.Args[0])
		os.Exit(1)
	}

	// The wrapped command we'll execute.
	cmd := os.Args[1]

	var cfg config
	if err := envdecode.StrictDecode(&cfg); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %s\n", err)
		os.Exit(1)
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
		fmt.Fprintf(os.Stderr, "fatal: %s\n", err)
		os.Exit(1)
	}
	defer out.Body.Close()

	env := make(map[string]string)
	if err = json.NewDecoder(out.Body).Decode(&env); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %s\n", err)
		os.Exit(1)
	}

	exe, err := exec.LookPath(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %s: %s\n", os.Args[1], err)
		os.Exit(1)
	}

	if err = syscall.Exec(exe, nil, toEnviron(env)); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %s: %s\n", exe, err)
		os.Exit(1)
	}
}

func toEnviron(env map[string]string) []string {
	result := make([]string, 0, len(env))

	for k, v := range env {
		result = append(result, k+"="+v)
	}

	return result
}
