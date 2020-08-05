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
	"github.com/alphagov/paas-prometheus-endpoints/pkg/service_plans_fetcher"

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

	cfg := config.NewConfigFromEnv("postgres")

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

	metricFetcher := NewExampleMetricFetcher(cfg.Logger)
	metricEndpoint := metric_endpoint.MetricEndpoint(servicePlansFetcher, metricFetcher, cfg.Logger)

	router := gin.Default()
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "online",
		})
	})
	auth := authenticator.NewBasicAuthenticator(cfg.CFClientConfig.ApiAddress, nil)
	authenticatedRoutes := router.Group("/")
	authenticatedRoutes.Use(authenticator.AuthenticatorMiddleware(auth, cfg.Logger))
	authenticatedRoutes.GET("/metrics", metricEndpoint)
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
