package config

import "fmt"

// LoadJobRotateKeys reads and validates the runtime configuration for the key-rotation job.
func LoadJobRotateKeys() (JobRotateKeys, error) {
	appConfig, err := LoadApp()
	if err != nil {
		return JobRotateKeys{}, fmt.Errorf("load rotate-keys config: %w", err)
	}

	return JobRotateKeys{
		App: Main{
			Name:      appConfig.App.Name + "-job-rotate-keys",
			MasterKey: appConfig.App.MasterKey,
		},
		Jwk:      JwkPresetDefault,
		Otel:     appConfig.Otel,
		Postgres: appConfig.Postgres,
	}, nil
}
