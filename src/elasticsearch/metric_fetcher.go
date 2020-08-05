package main

import (
	"fmt"
	// "net/http"
	"sync"

	aiven_service_discovery_writer "github.com/alphagov/paas-observability-release/src/aiven-service-discovery/pkg/writer"
	"github.com/alphagov/paas-prometheus-endpoints/pkg/authenticator"
	"github.com/alphagov/paas-prometheus-endpoints/pkg/metric_endpoint"

	"code.cloudfoundry.org/lager"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/gin-gonic/gin"
)

type ElasticsearchMetricFetcher struct {
	aivenProjectName        string
	aivenPrometheusUsername string
	aivenPrometheusPassword string

	logger lager.Logger

	aivenServicePrometheusTargets map[string]aiven_service_discovery_writer.PrometheusTargetConfig
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
	PrometheusTargetConfig aiven_service_discovery_writer.PrometheusTargetConfig
}

func (f *ElasticsearchMetricFetcher) FetchMetrics(
	c *gin.Context,
	user authenticator.User,
	serviceInstances []cfclient.ServiceInstance,
	servicePlans []cfclient.ServicePlan,
	service cfclient.Service,
) ([]metric_endpoint.Metric, error) {
	// logger := f.logger.Session("fetch-metrics", lager.Data{
	// 	"username": user.Username(),
	// })

	// prometheusTargets := f.findPrometheusEndpoints(serviceInstances, logger)

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

func (f *ElasticsearchMetricFetcher) WritePrometheusTargetConfigs(
	targets []aiven_service_discovery_writer.PrometheusTargetConfig,
) {
	aivenServicePrometheusTargets := map[string]aiven_service_discovery_writer.PrometheusTargetConfig{}
	for _, target := range targets {
		aivenServicePrometheusTargets[target.Labels.ServiceName] = target
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	f.aivenServicePrometheusTargets = aivenServicePrometheusTargets
}

// // FIXME: Should this acceptheader be simplified in case endpoint behaviour will change?
// const acceptHeader = `application/openmetrics-text; version=0.0.1,text/plain;version=0.0.4;q=0.5,*/*;q=0.1`
// const userAgent = `paas-prometheus-endpoints-elasticsearch`

// func scrapePrometheusTargets(targets []ServiceInstancePrometheusTargetConfig) ([]metric_endpoint.Metric, error) {
// 	for _, target := range targets {
// 		for _, nodeIP := range target.PrometheusTargetConfig.Targets {
// 			// FIXME: Fetch the `/metrics` endpoint over HTTPS with the basic auth creds
// 			// the parent object.
// 			// FIXME: Surely I can use a library for this!

// 			// Adapted from https://github.com/prometheus/prometheus/blob/f482c7bdd734ba012f6cedd68ddc24b1416efb35/scrape/scrape.go#L601
// 			// FIXME: Package this as a separate library under the same license,
// 			// see https://twitter.com/markspolakovs/status/1288953990533853189
// 			url := fmt.Sprintf("https://%s/metrics", nodeIP.String())
// 			req, err := http.NewRequest("GET", url, nil)
// 			if err != nil {
// 				return nil, err
// 			}
// 			req.Header.Add("Accept", acceptHeader)
// 			// FIXME: Support gzip since the originating code does?
// 			//req.Header.Add("Accept-Encoding", "gzip")
// 			req.Header.Set("User-Agent", userAgent)

// 			// FIXME: Perform request

// 			resp, err := s.client.Do(s.req.WithContext(ctx))
// 			defer resp.Body.Close()

// 			if resp.StatusCode != http.StatusOK {
// 				return nil, errors.Errorf("server returned HTTP status %s", resp.Status)
// 			}

// 			textParser := expfmt.TextParser{}
// 			metricFamilies, err := textParser.TextToMetricFamilies(resp.Body)
// 			if err != nil {
// 				// FIXME: Do something
// 				// Perhaps return a slice or errors, rather than eager failing all of these?
// 				return nil, err
// 			}

// 			return nil, nil
// 		}
// 	}
// }
