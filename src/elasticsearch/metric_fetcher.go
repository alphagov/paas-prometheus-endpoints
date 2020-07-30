package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/alphagov/paas-observability-release/src/aiven-service-discovery/pkg/writer"
	"github.com/alphagov/paas-prometheus-endpoints/pkg/authenticator"
	"github.com/alphagov/paas-prometheus-endpoints/pkg/metric_endpoint"

	"code.cloudfoundry.org/lager"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/elasticache"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/gin-gonic/gin"
)

type ElasticsearchMetricFetcher struct {
	aivenProjectName        string
	aivenPrometheusUsername string
	aivenPrometheusPassword string

	logger lager.Logger

	aivenServicePrometheusTargets map[string]writer.PrometheusTargetConfig
	mu                            sync.RWMutex
}

func NewElasticsearchMetricFetcher(
	aivenProjectName string,
	aivenPrometheusUsername string,
	aivenPrometheusPassword string,
	logger lager.Logger,
) *ElasticsearchMetricFetcher {
	logger = logger.Session("elasticsearch-metric-fetcher")
	return &ElasticsearchMetricFetcher{
		aivenProjectName:        aivenProjectName,
		aivenPrometheusUsername: aivenPrometheusUsername,
		aivenPrometheusPassword: aivenPrometheusPassword,

		logger: logger,
	}
}

type ServiceInstancePrometheusTargetConfig struct {
	ServiceInstance        cfclient.ServiceInstance
	PrometheusTargetConfig writer.PrometheusTargetConfig
}

func (f *ElasticsearchMetricFetcher) FetchMetrics(
	c *gin.Context,
	user authenticator.CFUser,
	serviceInstances []cfclient.ServiceInstance,
	servicePlans []cfclient.ServicePlan,
	service cfclient.Service,
) ([]metric_endpoint.Metric, error) {
	logger := f.logger.Session("fetch-metrics", lager.Data{
		"username": user.Username(),
	})

	prometheusTargets := f.findPrometheusEndpoints(serviceInstances)

	// FIXME: Scrape the `/metrics` endpoint for all those prometheus targets,
	// using the needed basic auth.

	// FIXME: Output metrics for all the endpoints with similar labels to
	// the existing redis endpoint
	metrics := []metric_endpoint.Metric{}
	return metrics, nil
}

func (f *ElasticsearchMetricFetcher) findPrometheusEndpoints(
	serviceInstances []cfclient.ServiceInstance,
	logger lager.Logger,
) []ServiceInstancePrometheusTargetConfig {
	f.mu.RLock()
	defer f.mu.RUnlock()

	prometheusTargets := []ServiceInstancePrometheusTargetConfig{}
	for _, serviceInstance := range serviceInstances {
		expectedAivenServiceName := fmt.Sprintf("%s-%s", f.aivenProjectName, serviceInstance.Guid)
		prometheusTargetConfig, ok := f.aivenServicePrometheusTargets[expectedAivenServiceName]
		if !ok {
			// FIXME: Expose some sort of metric for this?
			logger.Info("prometheus-endpoints-not-found", lager.Data{
				"service-instance-guid":                        serviceInstance.Guid,
				"number-of-prometheus-endpoint-configs-stored": len(f.aivenServicePrometheusTargets),
			})
			continue
		}
		prometheusTarget := ServiceInstancePrometheusTargetConfig{
			ServiceInstance:        serviceInstance,
			PrometheusTargetConfig: prometheusTargetConfig,
		}
		prometheusTargets = append(prometheusTargets, prometheusTarget)
	}
	return prometheusTargets
}

func (f *ElasticsearchMetricFetcher) WritePrometheusTargetConfigs(targets []writer.PrometheusTargetConfig) {
	aivenServicePrometheusTargets := map[string]writer.PrometheusTargetConfig{}
	for _, target := range targets {
		aivenServicePrometheusTargets[target.Labels.ServiceName] = aivenServicePrometheusTargets
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	f.aivenServicePrometheusTargets = aivenServicePrometheusTargets
}
