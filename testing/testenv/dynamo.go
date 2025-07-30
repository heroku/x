package testenv

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// NewDynamoService configures a local dynamo client connected to DYNAMODB_HOST
// or localhost:8000, which should be running aws-dynamodb-local.
func NewDynamoService(t *testing.T) *dynamodb.Client {
	t.Helper()

	host, err := getenv("DYNAMODB_HOST", "localhost")
	if err != nil {
		t.Skip(err.Error())
	}

	config := aws.Config{
		Credentials:  credentials.NewStaticCredentialsProvider("AKAAAAAAAAABBBBBBBBB", "t+GuXOHmzo1joFaJeu/abcdefghijklmabcdefghi", ""),
		BaseEndpoint: aws.String("http://" + host + ":8000/"),
		Region:       "testing",
	}
	return dynamodb.NewFromConfig(config)
}
