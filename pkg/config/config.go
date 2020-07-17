package config

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"code.cloudfoundry.org/lager"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
)

type Config struct {
	DeployEnv  string
	AWSRegion  string
	Logger     lager.Logger
	ListenPort uint

	CFClientConfig *cfclient.Config
	ServiceName    string

	ServicePlanUpdateSchedule time.Duration
}

func NewConfigFromEnv(defaultServiceName string) Config {
	return Config{
		DeployEnv:  getEnvWithDefaultString("DEPLOY_ENV", "dev"),
		AWSRegion:  os.Getenv("AWS_REGION"),
		Logger:     getDefaultLogger(),
		ListenPort: getEnvWithDefaultInt("PORT", 9299),

		CFClientConfig: &cfclient.Config{
			ApiAddress:        os.Getenv("CF_API_ADDRESS"),
			Username:          os.Getenv("CF_USERNAME"),
			Password:          os.Getenv("CF_PASSWORD"),
			ClientID:          os.Getenv("CF_CLIENT_ID"),
			ClientSecret:      os.Getenv("CF_CLIENT_SECRET"),
			SkipSslValidation: os.Getenv("CF_SKIP_SSL_VALIDATION") == "true",
			Token:             os.Getenv("CF_TOKEN"),
			UserAgent:         os.Getenv("CF_USER_AGENT"),
			HttpClient: &http.Client{
				Timeout: 30 * time.Second,
			},
		},
		ServiceName: getEnvWithDefaultString("SERVICE_NAME", defaultServiceName),

		ServicePlanUpdateSchedule: getEnvWithDefaultDuration("SERVICE_PLAN_UPDATE_SCHEDULE", 15*time.Minute),
	}
}

func getEnvWithDefaultDuration(k string, def time.Duration) time.Duration {
	v := getEnvWithDefaultString(k, "")
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		panic(err)
	}
	return d
}

func getEnvWithDefaultString(k string, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}

func getEnvWithDefaultInt(k string, def uint) uint {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	d, err := strconv.ParseUint(v, 10, 32)
	if err != nil {
		panic(err)
	}
	return uint(d)
}

func getDefaultLogger() lager.Logger {
	logger := lager.NewLogger("prometheus-endpoint")
	logLevel := lager.INFO
	if strings.ToLower(os.Getenv("LOG_LEVEL")) == "debug" {
		logLevel = lager.DEBUG
	}
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, logLevel))

	return logger
}
