package main

import (
	"os"

	generic_config "github.com/alphagov/paas-prometheus-endpoints/pkg/config"
)

type Config struct {
	generic_config.Config

	AivenProjectName        string
	AivenPrometheusUsername string
	AivenPrometheusPassword string
}

func NewCustomConfigFromEnv(defaultServiceName string) Config {
	return Config{
		Config: generic_config.NewConfigFromEnv(defaultServiceName),

		AivenProjectName:        os.Getenv("AIVEN_PROJECT_NAME"),
		AivenPrometheusUsername: os.Getenv("AIVEN_PROMETHEUS_USERNAME"),
		AivenPrometheusPassword: os.Getenv("AIVEN_PROMETHEUS_PASSWORD"),
	}
}
