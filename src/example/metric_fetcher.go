package main

import (
	"fmt"
	"time"

	"github.com/alphagov/paas-prometheus-endpoints/pkg/authenticator"
	"github.com/alphagov/paas-prometheus-endpoints/pkg/metric_endpoint"

	"code.cloudfoundry.org/lager"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/gin-gonic/gin"
)

type ExampleMetricFetcher struct {
	logger lager.Logger
}

func NewExampleMetricFetcher(logger lager.Logger) *ExampleMetricFetcher {
	logger = logger.Session("example-metric-fetcher")
	return &ExampleMetricFetcher{logger}
}

func (f *ExampleMetricFetcher) FetchMetrics(
	c *gin.Context,
	user authenticator.User,
	serviceInstances []cfclient.ServiceInstance,
	servicePlans []cfclient.ServicePlan,
	service cfclient.Service,
) ([]metric_endpoint.Metric, error) {
	logger := f.logger.WithData(lager.Data{"username": user.Username()})
	logger.Debug("fetch-metrics")

	// Export a metric saying how many seconds it is since the service was created
	metrics := []metric_endpoint.Metric{}
	for _, serviceInstance := range serviceInstances {
		createdAt, err := time.Parse(time.RFC3339, serviceInstance.CreatedAt)
		if err != nil {
			logger.Error("error-parsing-time", err, lager.Data{
				"service-instance-guid": serviceInstance.Guid,
				"created-at":            serviceInstance.CreatedAt,
			})
			return nil, fmt.Errorf("error generating metrics: %v", err)
		}

		age := time.Now().Sub(createdAt)
		// FIXME: Support units and add seconds here
		ageMetric := metric_endpoint.Metric{
			Name:  "service_age",
			Value: age.Seconds(),
			Tags: map[string]string{
				"service_instance_guid": serviceInstance.Guid,
				"service_instance_name": serviceInstance.Name,
				"space_guid":            serviceInstance.SpaceGuid,
			},
		}
		metrics = append(metrics, ageMetric)
	}
	return metrics, nil
}
