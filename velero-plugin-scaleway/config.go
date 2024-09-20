package main

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pkg/errors"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/sirupsen/logrus"
)

const userAgentPrefix = "velero-plugin-scaleway"

type configBuilder struct {
	log        logrus.FieldLogger
	opts       []func(*config.LoadOptions) error
	optsScwReq []scw.RequestOption
	credsFlag  bool
}

func newConfigBuilder(logger logrus.FieldLogger) *configBuilder {
	return &configBuilder{
		log: logger,
	}
}

func newS3Client(cfg aws.Config, url string, forcePathStyle bool) (*s3.Client, error) {
	opts := []func(*s3.Options){
		func(o *s3.Options) {
			o.UsePathStyle = forcePathStyle
		},
	}
	if url != "" {
		if !IsValidS3URLScheme(url) {
			return nil, errors.Errorf("Invalid s3 url %s, URL must be valid according to https://golang.org/pkg/net/url/#Parse and start with http:// or https://", url)
		}
		opts = append(opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(url)
		})
	}

	return s3.NewFromConfig(cfg, opts...), nil
}

func (cb *configBuilder) WithSCWCredentials() *configBuilder {
	// Create a new provider, specify the environment variable names that store your credentials
	provider := NewScalewayCredentialsProvider("SCW_ACCESS_KEY", "SCW_SECRET_KEY")
	cb.opts = append(cb.opts, config.WithCredentialsProvider(provider))
	return cb
}

func (cb *configBuilder) WithSCWURL() *configBuilder {
	resolver := NewScalewayEndpointResolver("SCW_S3_ENDPOINT", "SCW_REGION")
	cb.opts = append(cb.opts, config.WithEndpointResolverWithOptions(resolver))

	return cb
}

func (cb *configBuilder) WithRegion(region string) *configBuilder {
	cb.optsScwReq = append(
		cb.optsScwReq,
		scw.WithRegions([]scw.Region{scw.Region(region)}...),
	)
	return cb
}

func (cb *configBuilder) WithZones(zone string) *configBuilder {
	cb.optsScwReq = append(
		cb.optsScwReq,
		scw.WithZones([]scw.Zone{scw.Zone(zone)}...),
	)

	return cb
}

func (cb *configBuilder) WithAuthRequest(accessKey, secretKey string) *configBuilder {
	cb.optsScwReq = append(cb.optsScwReq, scw.WithAuthRequest(accessKey, secretKey))

	return cb
}

func (cb *configBuilder) Build() (aws.Config, error) {
	conf, err := config.LoadDefaultConfig(context.Background(), cb.opts...)
	if err != nil {
		return aws.Config{}, err
	}
	if cb.credsFlag {
		if _, err := conf.Credentials.Retrieve(context.Background()); err != nil {
			return aws.Config{}, errors.WithStack(err)
		}
	}
	return conf, nil
}
