package main

import (
	"fmt"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

// ClientSCWError define an error when failing to create client
type ClientSCWError struct {
	Err     error
	Details string
}

func (s *ClientSCWError) Error() string {
	return s.Err.Error()
}

// configErrorDetails generate a detailed error message for an invalid client option.
func configErrorDetails(configKey, varEnv string) string {
	// TODO: update the more info link
	return fmt.Sprintf(`%s can be initialised using the command "scw init".

After initialisation, there are three ways to provide %s:
- with the Scaleway config file, in the %s key: %s;
- with the %s environement variable;

Note that the last method has the highest priority.

More info: https://github.com/scaleway/scaleway-sdk-go/tree/master/scw#scaleway-config`,
		configKey,
		configKey,
		configKey,
		scw.GetConfigPath(),
		varEnv,
	)
}
