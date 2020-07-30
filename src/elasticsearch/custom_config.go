package config

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	generic_config "github.com/alphagov/paas-prometheus-endpoints/pkg/config"

	"code.cloudfoundry.org/lager"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
)

type Config struct {
	generic_config.Config

	AivenProjectName        string
	AivenPrometheusUsername string
	AivenPrometheusPassword string
}

func NewCustomConfigFromEnv(defaultServiceName string) Config {
	return Config{
		generic_config.NewConfigFromEnv(defaultServiceName),

		AivenProjectName:        os.Getenv("AIVEN_PROJECT_NAME"),
		AivenPrometheusUsername: os.Getenv("AIVEN_PROMETHEUS_USERNAME"),
		AivenPrometheusPassword: os.Getenv("AIVEN_PROMETHEUS_PASSWORD"),
	}
}
