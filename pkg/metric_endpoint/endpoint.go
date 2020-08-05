package metric_endpoint

import (
	"net/http"

	"github.com/alphagov/paas-prometheus-endpoints/pkg/authenticator"
	"github.com/alphagov/paas-prometheus-endpoints/pkg/service_plans_fetcher"

	"code.cloudfoundry.org/lager"
	"github.com/gin-gonic/gin"
)

func MetricEndpoint(
	servicePlansFetcher *service_plans_fetcher.ServicePlansFetcher,
	serviceMetricsFetcher ServiceMetricFetcher,
	logger lager.Logger,
) gin.HandlerFunc {
	logger = logger.Session("metric-endpoint")

	return func(c *gin.Context) {
		user := c.MustGet("authenticated_user").(authenticator.User)

		service, ok := servicePlansFetcher.GetService()
		if !ok {
			logger.Error("service not found", nil)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"message": "an error occurred when trying to fetch the service",
			})
			return
		}
		servicePlans := servicePlansFetcher.GetServicePlans()
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

		serviceMetrics, err := serviceMetricsFetcher.FetchMetrics(c, user, serviceInstances, servicePlans, *service)
		if err != nil {
			logger.Error("error fetching service metrics", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"message": "an error occurred when fetching metrics for your service instances",
			})
			return
		}

		groupedServiceMetrics := groupMetricsByName(serviceMetrics)
		renderedOutput := ""
		for metricName, metricGroup := range groupedServiceMetrics {
			renderedOutput += renderMetricGroup(metricName, metricGroup)
		}

		c.Data(http.StatusOK, "text/plain; version=0.0.4", []byte(renderedOutput))
	}
}
