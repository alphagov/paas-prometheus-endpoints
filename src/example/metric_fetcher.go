package main

import (
	"fmt"
	"time"

	"github.com/alphagov/paas-prometheus-endpoints/pkg/authenticator"
	"github.com/alphagov/paas-prometheus-endpoints/pkg/metric_endpoint"

	"code.cloudfoundry.org/lager"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/gin-gonic/gin"
	dto "github.com/prometheus/client_model/go"
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
) (metric_endpoint.Metrics, error) {
	logger := f.logger.WithData(lager.Data{"username": user.Username()})
	logger.Debug("fetch-metrics")

	// Export a metric saying how many seconds it is since the service was created
	ageMetrics := []*dto.Metric{}
	for _, serviceInstance := range serviceInstances {
		ageMetric, err := fetchServiceInstanceAgeMetric(serviceInstance)
		if err != nil {
			// FIXME: Log a metric so we can alert on errors
			logger.Error("error-fetching-age-metric", err, lager.Data{
				"service-instance-guid": serviceInstance.Guid,
				"created-at":            serviceInstance.CreatedAt,
			})
			continue
		}
		ageMetrics = append(ageMetrics, ageMetric)
	}
	return metric_endpoint.Metrics{
		"service_age_seconds": &dto.MetricFamily{
			Name:   derefS("service_age_seconds"),
			Type:   derefT(dto.MetricType_GAUGE),
			Metric: ageMetrics,
		},
	}, nil
}

func fetchServiceInstanceAgeMetric(
	serviceInstance cfclient.ServiceInstance,
) (*dto.Metric, error) {
	createdAt, err := time.Parse(time.RFC3339, serviceInstance.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("error parsing time: %v", err)
	}

	age := time.Now().Sub(createdAt).Seconds()
	return &dto.Metric{
		Gauge: &dto.Gauge{
			Value: &age,
		},
		Label: []*dto.LabelPair{
			{
				Name:  derefS("service_instance_name"),
				Value: derefS(serviceInstance.Name),
			},
			{
				Name:  derefS("service_instance_guid"),
				Value: derefS(serviceInstance.Guid),
			},
			{
				Name:  derefS("space_guid"),
				Value: derefS(serviceInstance.SpaceGuid),
			},
		},
	}, nil
}

func derefS(s string) *string {
	return &s
}

func derefT(i dto.MetricType) *dto.MetricType {
	return &i
}
