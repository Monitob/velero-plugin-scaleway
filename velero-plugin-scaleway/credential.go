package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

// ScalewayCredentialsProvider implements the CredentialsProvider interface
type ScalewayCredentialsProvider struct {
	accessKeyEnvVar string
	secretKeyEnvVar string
}

// NewScalewayCredentialsProvider creates a new ScalewayCredentialsProvider
func NewScalewayCredentialsProvider(accessKeyEnvVar, secretKeyEnvVar string) *ScalewayCredentialsProvider {
	return &ScalewayCredentialsProvider{
		accessKeyEnvVar: accessKeyEnvVar,
		secretKeyEnvVar: secretKeyEnvVar,
	}
}

// Retrieve retrieves the credentials from environment variables or another source.
func (scp *ScalewayCredentialsProvider) Retrieve(ctx context.Context) (aws.Credentials, error) {
	accessKey := os.Getenv(scp.accessKeyEnvVar)
	secretKey := os.Getenv(scp.secretKeyEnvVar)

	if accessKey == "" || secretKey == "" {
		return aws.Credentials{}, fmt.Errorf("credentials not available from environment variables")
	}

	// Returning AWS-compatible credentials
	return aws.Credentials{
		AccessKeyID:     accessKey,
		SecretAccessKey: secretKey,
		// AWS requires this for credentials validity tracking, so we can use `false` since
		// Scaleway credentials don't expire in this way.
		CanExpire: false,
	}, nil
}

// WithCredentialsProvider takes an AWS-style credentials provider and returns an option function for loading credentials.
func WithCredentialsProvider(v aws.CredentialsProvider) func(option *scw.ClientOption) error {
	return func(o *scw.ClientOption) error {
		// Use AWS credentials provider in the options (adjust this for your needs)
		creds, err := v.Retrieve(context.Background())
		if err != nil {
			return fmt.Errorf("failed to retrieve credentials: %w", err)
		}
		// Use the credentials for Scaleway client options
		*o = scw.WithAuth(creds.AccessKeyID, creds.SecretAccessKey)
		return nil
	}
}
