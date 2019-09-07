package testenv

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

// NewDynamoService configures a local dynamo client connected to DYNAMODB_HOST
// or localhost:8000, which should be running aws-dynamodb-local.
func NewDynamoService(t *testing.T) *dynamodb.DynamoDB {
	t.Helper()

	host, err := getenv("DYNAMODB_HOST", "localhost")
	if err != nil {
		t.Skip(err.Error())
	}

	config := &aws.Config{
		Credentials: credentials.NewStaticCredentialsFromCreds(credentials.Value{
			AccessKeyID:     "AKAAAAAAAAABBBBBBBBB",
			SecretAccessKey: "t+GuXOHmzo1joFaJeu/abcdefghijklmabcdefghi",
		}),
		Endpoint: aws.String("http://" + host + ":8000/"),
		Region:   aws.String("testing"),
	}
	sess, err := session.NewSession(config)
	if err != nil {
		t.Fatal("unexpected error", err)
	}
	return dynamodb.New(sess)
}
