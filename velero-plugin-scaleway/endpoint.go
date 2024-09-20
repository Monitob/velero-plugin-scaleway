package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"os"
)

// ScalewayEndpointResolver implements the EndpointResolverWithOptions interface for Scaleway.
type ScalewayEndpointResolver struct {
	// Add fields for configuration if needed.
	urlEnvVar string
	region    string
}

// NewScalewayEndpointResolver initializes a new ScalewayEndpointResolver.
func NewScalewayEndpointResolver(urlEnvVar, regionEnvVar string) *ScalewayEndpointResolver {
	return &ScalewayEndpointResolver{
		urlEnvVar: urlEnvVar,
		region:    regionEnvVar,
	}
}

// ResolveEndpoint resolves the endpoint for a Scaleway service in a given region.
func (r *ScalewayEndpointResolver) ResolveEndpoint(service, region string, options ...interface{}) (aws.Endpoint, error) {
	// Define available regions for Scaleway
	validRegions := map[string]bool{
		"fr-par": true, // Paris, France
		"nl-ams": true, // Amsterdam, Netherlands
		"pl-waw": true, // Warsaw, Poland
	}

	// Check if the region is valid.
	if !validRegions[region] {
		return aws.Endpoint{}, fmt.Errorf("region %s is not supported for Scaleway", region)
	}

	var url string
	switch service {
	case "s3":
		customEndpoint := os.Getenv(r.urlEnvVar)
		if customEndpoint != "" {
			url = customEndpoint // Use the custom endpoint if it's set.
		} else {
			// Fallback to default S3 endpoint format.
			url = fmt.Sprintf("https://s3.%s.scw.cloud", region)
		}
	default:
		return aws.Endpoint{}, fmt.Errorf("service %s is not supported", service)
	}

	// Return the constructed endpoint.
	return aws.Endpoint{
		URL:           url,
		SigningRegion: region,
	}, nil
}
