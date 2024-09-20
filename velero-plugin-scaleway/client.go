package main

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/scaleway/scaleway-sdk-go/logger"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/scaleway-sdk-go/validation"
	"github.com/sirupsen/logrus"
	"strings"
)

// client Builder
type clientBuilder struct {
	log  logrus.FieldLogger
	opts []scw.ClientOption
}

func newClientBuilder(logger logrus.FieldLogger) *clientBuilder {
	return &clientBuilder{
		log: logger,
	}
}

func (c *clientBuilder) WithEnvProfile() *clientBuilder {
	c.opts = append(c.opts, scw.WithEnv())

	return c
}

func (c *clientBuilder) WithUserAgent(userAgent string) *clientBuilder {
	c.opts = append(c.opts, scw.WithUserAgent(userAgent))

	return c
}

func (c *clientBuilder) WithRegion(region string) *clientBuilder {
	c.opts = append(c.opts, scw.WithDefaultRegion(scw.Region(region)))

	return c
}

func (c *clientBuilder) Build(configPath string, profileName string) (*scw.Client, error) {
	profile := scw.LoadEnvProfile()

	// Default path is based on the following priority order:
	// * The config file's path provided via --config flag
	// * $SCW_CONFIG_PATH
	// * $XDG_CONFIG_HOME/scw/config.yaml
	// * $HOME/.config/scw/config.yaml
	// * $USERPROFILE/.config/scw/config.yaml
	configFromPath, err := scw.LoadConfigFromPath(configPath)
	switch {
	case errIsConfigFileNotFound(err):
		// no config file was found -> nop

	case err != nil:
		// failed to read the config file -> fail
		return nil, err

	default:
		// found and loaded a config file -> merge with env
		activeProfile, err := configFromPath.GetProfile(profileName)
		if err != nil {
			return nil, err
		}

		// Creates a client from the active profile
		// It will trigger a validation step on its configuration to catch errors if any
		opts := []scw.ClientOption{
			scw.WithProfile(activeProfile),
		}

		_, err = scw.NewClient(opts...)
		if err != nil {
			return nil, err
		}

		profile = scw.MergeProfiles(activeProfile, profile)
	}

	// If profile have a defaultZone but no defaultRegion we set the defaultRegion
	// to the one of the defaultZone
	if profile.DefaultZone != nil && *profile.DefaultZone != "" &&
		(profile.DefaultRegion == nil || *profile.DefaultRegion == "") {
		zone := *profile.DefaultZone
		logger.Debugf("guess region from %s zone", zone)
		region := zone[:len(zone)-2]
		if validation.IsRegion(region) {
			profile.DefaultRegion = scw.StringPtr(region)
		} else {
			logger.Debugf("invalid guessed region '%s'", region)
		}
	}

	client, err := scw.NewClient(c.opts...)
	if err != nil {
		return nil, err
	}

	return client, validateClient(client)
}

// validateClient validate a client configuration and make sure all mandatory setting are present.
// This function is only call for commands that require a valid client.
func validateClient(client *scw.Client) error {
	accessKey, _ := client.GetAccessKey()
	if accessKey == "" {
		return &ClientSCWError{
			Err:     fmt.Errorf("access key is required"),
			Details: configErrorDetails("access_key", "SCW_ACCESS_KEY"),
		}
	}

	if !validation.IsAccessKey(accessKey) {
		return &ClientSCWError{
			Err: fmt.Errorf("invalid access key format '%s', expected SCWXXXXXXXXXXXXXXXXX format", accessKey),
		}
	}

	secretKey, _ := client.GetSecretKey()
	if secretKey == "" {
		return &ClientSCWError{
			Err:     fmt.Errorf("secret key is required"),
			Details: configErrorDetails("secret_key", "SCW_SECRET_KEY"),
		}
	}

	if !validation.IsSecretKey(secretKey) {
		return &ClientSCWError{
			Err: fmt.Errorf("invalid secret key format '%s', expected a UUID: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx", secretKey),
		}
	}

	defaultOrganizationID, _ := client.GetDefaultOrganizationID()
	if defaultOrganizationID == "" {
		return &ClientSCWError{
			Err:     fmt.Errorf("organization ID is required"),
			Details: configErrorDetails("default_organization_id", "SCW_DEFAULT_ORGANIZATION_ID"),
		}
	}

	if !validation.IsOrganizationID(defaultOrganizationID) {
		return &ClientSCWError{
			Err: fmt.Errorf("invalid organization ID format '%s', expected a UUID: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx", defaultOrganizationID),
		}
	}

	defaultZone, _ := client.GetDefaultZone()
	if defaultZone == "" {
		return &ClientSCWError{
			Err:     fmt.Errorf("default zone is required"),
			Details: configErrorDetails("default_zone", "SCW_DEFAULT_ZONE"),
		}
	}

	if !validation.IsZone(defaultZone.String()) {
		zones := []string(nil)
		for _, z := range scw.AllZones {
			zones = append(zones, string(z))
		}
		return &ClientSCWError{
			Err: fmt.Errorf("invalid default zone format '%s', available zones are: %s", defaultZone, strings.Join(zones, ", ")),
		}
	}

	defaultRegion, _ := client.GetDefaultRegion()
	if defaultRegion == "" {
		return &ClientSCWError{
			Err:     fmt.Errorf("default region is required"),
			Details: configErrorDetails("default_region", "SCW_DEFAULT_REGION"),
		}
	}

	if !validation.IsRegion(defaultRegion.String()) {
		regions := []string(nil)
		for _, z := range scw.AllRegions {
			regions = append(regions, string(z))
		}
		return &ClientSCWError{
			Err: fmt.Errorf("invalid default region format '%s', available regions are: %s", defaultRegion, strings.Join(regions, ", ")),
		}
	}

	return nil
}

func errIsConfigFileNotFound(err error) bool {
	var target *scw.ConfigFileNotFoundError
	return errors.As(err, &target)
}
