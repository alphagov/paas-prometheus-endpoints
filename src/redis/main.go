package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/alphagov/paas-prometheus-endpoints/pkg/authenticator"
	"github.com/alphagov/paas-prometheus-endpoints/pkg/config"
	"github.com/alphagov/paas-prometheus-endpoints/pkg/metric_endpoint"
	"github.com/alphagov/paas-prometheus-endpoints/pkg/orgs_fetcher"
	"github.com/alphagov/paas-prometheus-endpoints/pkg/service_plans_fetcher"
	"github.com/alphagov/paas-prometheus-endpoints/pkg/spaces_fetcher"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/elasticache"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/gin-gonic/gin"
)

func main() {
	ctx, shutdown := context.WithCancel(context.Background())
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Reset(syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		shutdown()
	}()
	var wg sync.WaitGroup

	cfg := config.NewConfigFromEnv("redis")

	cfClient, err := cfclient.NewClient(cfg.CFClientConfig)
	if err != nil {
		if err != nil {
			cfg.Logger.Error("err-unable-to-initialise-own-cf-client", err)
		}
		shutdown()
		os.Exit(1)
	}

	servicePlansFetcher := service_plans_fetcher.NewServicePlansFetcher(cfg.ServiceName, cfg.ServicePlanUpdateSchedule, cfg.Logger, cfClient)
	wg.Add(1)
	go func() {
		err := servicePlansFetcher.Run(ctx)
		if err != nil {
			cfg.Logger.Error("err-fatal-service-plans-fetcher", err)
		}
		shutdown()
		os.Exit(1)
	}()

	spacesFetcher := spaces_fetcher.NewSpacesFetcher(cfg.SpaceUpdateSchedule, cfg.Logger, cfClient)
	wg.Add(1)
	go func() {
		err := spacesFetcher.Run(ctx)
		if err != nil {
			cfg.Logger.Error("err-fatal-spaces-fetcher", err)
		}
		shutdown()
		os.Exit(1)
	}()

	orgsFetcher := orgs_fetcher.NewOrgsFetcher(cfg.OrgUpdateSchedule, cfg.Logger, cfClient)
	wg.Add(1)
	go func() {
		err := orgsFetcher.Run(ctx)
		if err != nil {
			cfg.Logger.Error("err-fatal-orgs-fetcher", err)
		}
		shutdown()
		os.Exit(1)
	}()

	awsConfig := aws.NewConfig().WithRegion(cfg.AWSRegion)
	awsSession := session.Must(session.NewSession(awsConfig))
	elasticacheClient := elasticache.New(awsSession)
	cloudwatchClient := cloudwatch.New(awsSession)

	redisMetricFetcher := NewRedisMetricFetcher(elasticacheClient, cloudwatchClient, cfg.Logger)
	redisMetricEndpoint := metric_endpoint.MetricEndpoint(servicePlansFetcher, spacesFetcher, orgsFetcher, redisMetricFetcher, cfg.Logger)

	router := gin.Default()
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "online",
		})
	})
	auth := authenticator.NewBasicAuthenticator(cfg.CFClientConfig.ApiAddress, nil)
	authenticatedRoutes := router.Group("/")
	authenticatedRoutes.Use(authenticator.AuthenticatorMiddleware(auth, cfg.Logger))
	authenticatedRoutes.GET("/metrics", redisMetricEndpoint)
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.ListenPort),
		Handler: router,
	}

	wg.Add(1)
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			cfg.Logger.Error("err-fatal-server", err)
		}
		shutdown()
		os.Exit(1)
	}()

	wg.Wait()
}
