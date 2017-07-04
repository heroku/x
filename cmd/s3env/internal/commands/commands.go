/*
NAME
        s3env - manage ENV vars in an S3 object

PREREQUISITES
        The following ENV vars are required to use s3env:

	- S3ENV_KEY (defaults to env.json)
	- S3ENV_BUCKET
	- S3ENV_AWS_ACCESS_KEY_ID
	- S3ENV_AWS_REGION
	- S3ENV_AWS_SECRET_ACCESS_KEY

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
*/
package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/joeshaw/envdecode"
	"github.com/spf13/cobra"
)

var (
	outputShell bool
	outputJSON  bool
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "s3env <command> [FLAGS]",
	Short: "s3env manages config vars and stores them on an s3 object.",
	Long: `s3env wraps an existing command and sets ENV vars for them.

NOTE: These ENV vars are necessary for using s3env:

        S3ENV_KEY (defaults to env.json)
        S3ENV_BUCKET
        S3ENV_AWS_ACCESS_KEY_ID
        S3ENV_AWS_REGION
        S3ENV_AWS_SECRET_ACCESS_KEY
`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		// fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(loadS3Object)

	RootCmd.PersistentFlags().BoolVarP(&outputShell, "shell", "s", true, "output config vars in shell format")
	RootCmd.PersistentFlags().BoolVarP(&outputJSON, "json", "", false, "output config vars in JSON format")
	RootCmd.AddCommand(configCmd)
	RootCmd.AddCommand(configGetCmd)
	RootCmd.AddCommand(configSetCmd)
	RootCmd.AddCommand(configUnsetCmd)
	RootCmd.AddCommand(runCmd)
}

func displayUsage(cmd *cobra.Command) {
	fmt.Fprintln(os.Stderr, "Usage: s3env "+cmd.Use)
	os.Exit(1)
}

func displayErr(err error) {
	fmt.Fprintln(os.Stderr, "s3env: "+err.Error())
	os.Exit(1)
}

var (
	s3vars = make(map[string]string)
	cfg    config
	client s3iface.S3API
)

func loadS3Object() {
	if err := envdecode.StrictDecode(&cfg); err != nil {
		fmt.Printf("s3env: %s (continuing with empty config)\n", err)
		return
	}

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(cfg.Region),
		Credentials: credentials.NewStaticCredentials(
			cfg.AccessID,
			cfg.SecretKey,
			"",
		),
	})

	if err != nil {
		log.Fatalf("getting aws session failed: %s", err)
	}

	client = s3.New(sess)

	in, err := input()
	if err != nil {
		log.Fatalf("error reading input: %s", err)
	}
	defer in.Close()

	if err = json.NewDecoder(in).Decode(&s3vars); err != nil {
		log.Fatalf("error decoding json: %s", err)
	}
}

func displayVars(vars map[string]string) {
	if outputJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.Encode(vars)
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
			return nil, fmt.Errorf("Unable to parse %s. Make sure it's of the format KEY=VAL", v)
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

	_, err := client.PutObject(&s3.PutObjectInput{
		Bucket:               aws.String(cfg.Bucket),
		Key:                  aws.String(cfg.Key),
		Body:                 bytes.NewReader(buf.Bytes()),
		ServerSideEncryption: aws.String(cfg.ServerSideEncryption),
	})

	if err != nil {
		return fmt.Errorf("saving to s3 failed with error: %s", err)
	}
	return nil
}

// config is used in production when you want to fetch a JSON config
// from an S3 object.
type config struct {
	Key    string `env:"S3ENV_KEY,default=env.json"`
	Bucket string `env:"S3ENV_BUCKET,required"`

	AccessID  string `env:"S3ENV_AWS_ACCESS_KEY_ID,required"`
	Region    string `env:"S3ENV_AWS_REGION,required"`
	SecretKey string `env:"S3ENV_AWS_SECRET_ACCESS_KEY,required"`

	ServerSideEncryption string `env:"S3ENV_AWS_SERVER_SIDE_ENCRYPTION,default=AES256"`
}

// input gets the appropriate input source. If there was any data pumped into
// STDIN, we'll choose that. Otherwise we'll try to load the s3 object that
// was configured.
func input() (io.ReadCloser, error) {
	out, err := client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(cfg.Bucket),
		Key:    aws.String(cfg.Key),
	})
	if err != nil {
		// Cast err to awserr.Error to handle specific error codes.
		aerr, ok := err.(awserr.Error)
		if ok && aerr.Code() == s3.ErrCodeNoSuchKey {
			fmt.Fprintf(os.Stderr, "s3env: object not found. using empty config\n")
			buf := new(bytes.Buffer)
			buf.Write([]byte("{}"))

			return ioutil.NopCloser(buf), nil
		}
		return nil, err
	}
	fmt.Fprintf(os.Stderr, "s3env: object %s/%s found\n", cfg.Bucket, cfg.Key)
	return out.Body, nil
}
