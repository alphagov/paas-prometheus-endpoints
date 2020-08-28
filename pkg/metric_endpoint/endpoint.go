package metric_endpoint

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/alphagov/paas-prometheus-endpoints/pkg/authenticator"
	"github.com/alphagov/paas-prometheus-endpoints/pkg/service_plans_fetcher"

	"code.cloudfoundry.org/lager"
	"github.com/gin-gonic/gin"
)

func MetricEndpoint(
	servicePlansStore service_plans_fetcher.ServicePlansStore,
	serviceMetricsFetcher ServiceMetricFetcher,
	logger lager.Logger,
) gin.HandlerFunc {
	logger = logger.Session("metric-endpoint")

	return func(c *gin.Context) {
		user := c.MustGet("authenticated_user").(authenticator.User)

		metrics, err := GetMetricsForUser(user, servicePlansStore, serviceMetricsFetcher, c, logger)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}

		output := &bytes.Buffer{}
		contentLength := renderMetricsInPromFormat(metrics, output, logger)
		c.DataFromReader(
			http.StatusOK,
			int64(contentLength),
			"text/plain; version=0.0.4",
			output,
			nil,
		)
	}
}

func GetMetricsForUser(
	user authenticator.User,
	servicePlansStore service_plans_fetcher.ServicePlansStore,
	serviceMetricsFetcher ServiceMetricFetcher,
	c *gin.Context,
	logger lager.Logger,
) (Metrics, error) {
	service := servicePlansStore.GetService()
	if service == nil {
		logger.Error("err-service-not-found", nil)
		return nil, fmt.Errorf("an error occurred when trying to fetch the service")
	}

	servicePlans := servicePlansStore.GetServicePlans()
	servicePlanGUIDs := make([]string, len(servicePlans))
	for i, servicePlan := range servicePlans {
		servicePlanGUIDs[i] = servicePlan.Guid
	}

	serviceInstances, err := user.ListServiceInstancesMatchingPlanGUIDs(servicePlanGUIDs)
	if err != nil {
		logger.Error("err-listing-service-instances", err)
		return nil, fmt.Errorf("an error occurred when trying to list your service instances")
	}

	metrics, err := serviceMetricsFetcher.FetchMetrics(c, user, serviceInstances, servicePlans, *service)
	if err != nil {
		logger.Error("err-fetching-service-metrics", err)
		return nil, fmt.Errorf("an error occurred when fetching metrics for your service instances")
	}

	return metrics, nil
}
