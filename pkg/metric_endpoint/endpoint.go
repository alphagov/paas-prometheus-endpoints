package metric_endpoint

import (
	"bytes"
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

		service := servicePlansStore.GetService()
		if service == nil {
			logger.Error("service not found", nil)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"message": "an error occurred when trying to fetch the service",
			})
			return
		}
		servicePlans := servicePlansStore.GetServicePlans()
		servicePlanGUIDs := make([]string, len(servicePlans))
		for i, servicePlan := range servicePlans {
			servicePlanGUIDs[i] = servicePlan.Guid
		}

		serviceInstances, err := user.ListServiceInstancesMatchingPlanGUIDs(servicePlanGUIDs)
		if err != nil {
			logger.Error("error listing service instances", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"message": "an error occurred when trying to list your service instances",
			})
			return
		}

		metrics, err := serviceMetricsFetcher.FetchMetrics(c, user, serviceInstances, servicePlans, *service)
		if err != nil {
			logger.Error("error fetching service metrics", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"message": "an error occurred when fetching metrics for your service instances",
			})
			return
		}

		output := &bytes.Buffer{}
		contentLength := renderMetrics(metrics, output, logger)
		c.DataFromReader(
			http.StatusOK,
			int64(contentLength),
			"text/plain; version=0.0.4",
			output,
			nil,
		)
	}
}
