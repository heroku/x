package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pkg/errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/joeshaw/envdecode"
	"github.com/spf13/cobra"
)

func init() {
	cobra.OnInitialize(loadS3Object)

	Root.PersistentFlags().BoolVarP(&outputShell, "shell", "s", true, "output config vars in shell format")
	Root.PersistentFlags().BoolVarP(&outputJSON, "json", "", false, "output config vars in JSON format")
	Root.AddCommand(configCmd)
	Root.AddCommand(configGetCmd)
	Root.AddCommand(configSetCmd)
	Root.AddCommand(configUnsetCmd)
	Root.AddCommand(runCmd)
}

func main() {
	if err := Root.Execute(); err != nil {
		os.Exit(1)
	}
}

var (
	outputShell bool
	outputJSON  bool
	s3vars      = make(map[string]string)
	cfg         config
	client      interface {
		GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)
		PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	}
)

// Root represents the base command when called without any subcommands
var Root = &cobra.Command{
	Use:   "s3env <command> [FLAGS]",
	Short: "s3env manages config vars and stores them on an s3 object.",
	Long: `s3env wraps an existing command and sets ENV vars for them.

PREREQUISITES
	The following ENV vars are required to use s3env:

        S3ENV_KEY (defaults to env.json)
        S3ENV_BUCKET
        S3ENV_AWS_ACCESS_KEY_ID
        S3ENV_AWS_REGION
		S3ENV_AWS_SECRET_ACCESS_KEY

EXAMPLES
        s3env config                  # show all config vars
        s3env config:set FOO=1 BAR=2  # set two vars
        s3env config:get FOO          # display FOO
        s3env config:unset FOO        # remove FOO
        s3env run hello-world         # hello-world will get BAR=2 defined in its ENV

CONTEXT
        One of the limitations of heroku config vars presently is the total
        size you can configure on any given app (32kb). If you're managing
        lots of TLS certificates, that limit quickly runs out.

`,
}

func displayUsage(cmd *cobra.Command) {
	fmt.Fprintln(os.Stderr, "Usage: s3env "+cmd.Use)
	os.Exit(1)
}

func displayErr(err error) {
	fmt.Fprintln(os.Stderr, "s3env: "+err.Error())
	os.Exit(1)
}

func loadS3Object() {
	if err := envdecode.StrictDecode(&cfg); err != nil {
		fmt.Printf("s3env: %s (continuing with empty config)\n", err)
		return
	}

	client = s3.NewFromConfig(aws.Config{
		Region: cfg.Region,
		Credentials: credentials.NewStaticCredentialsProvider(
			cfg.AccessID,
			cfg.SecretKey,
			"",
		),
	})

	in, err := input()
	if err != nil {
		fmt.Printf("s3env: read input error: %s\n", err)
		return
	}
	defer in.Close()

	if err = json.NewDecoder(in).Decode(&s3vars); err != nil {
		fmt.Printf("s3env: decode input error: %s\n", err)
		return
	}
}

func displayVars(vars map[string]string) {
	if outputJSON {
		enc := json.NewEncoder(os.Stdout)
		if err := enc.Encode(vars); err != nil {
			panic(err)
		}
		return
	}

	for k, v := range vars {
		if strings.Contains(v, "\n") {
			fmt.Printf("%s='%s'\n", k, v)
		} else {
			fmt.Printf("%s=%s\n", k, v)
		}
	}
}

func parseEnvironStrings(environ []string) (map[string]string, error) {
	vars := make(map[string]string)

	for _, v := range environ {
		chunks := strings.SplitN(v, "=", 2)
		if len(chunks) != 2 {
			return nil, fmt.Errorf("unable to parse %q (expected format KEY=VAL)", v)
		}

		vars[chunks[0]] = chunks[1]
	}

	return vars, nil
}

func persistVars() error {
	var buf bytes.Buffer

	if err := json.NewEncoder(&buf).Encode(s3vars); err != nil {
		return fmt.Errorf("encode failed: %s", err)
	}

	sse := s3types.ServerSideEncryption(cfg.ServerSideEncryption)
	switch sse {
	case s3types.ServerSideEncryptionAes256,
		s3types.ServerSideEncryptionAwsFsx,
		s3types.ServerSideEncryptionAwsKms,
		s3types.ServerSideEncryptionAwsKmsDsse:
	default:
		return fmt.Errorf("unrecognized value for S3ENV_AWS_SERVER_SIDE_ENCRYPTION: %s", cfg.ServerSideEncryption)
	}
	_, err := client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:               aws.String(cfg.Bucket),
		Key:                  aws.String(cfg.Key),
		Body:                 bytes.NewReader(buf.Bytes()),
		ServerSideEncryption: sse,
	})
	if err != nil {
		return fmt.Errorf("saving to s3 failed with error: %s", err)
	}
	return nil
}

// input gets the appropriate input source. If there was any data pumped into
// STDIN, we'll choose that. Otherwise we'll try to load the s3 object that
// was configured.
func input() (io.ReadCloser, error) {
	out, err := client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(cfg.Bucket),
		Key:    aws.String(cfg.Key),
	})
	if err != nil {
		nsk := &s3types.NoSuchKey{}
		if errors.As(err, &nsk) {
			fmt.Fprintf(os.Stderr, "s3env: object not found. using empty config\n")
			buf := new(bytes.Buffer)
			buf.Write([]byte("{}"))

			return io.NopCloser(buf), nil
		}
		return nil, err
	}
	return out.Body, nil
}
