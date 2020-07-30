package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	aiven_service_discovery_writer "github.com/alphagov/paas-observability-release/src/aiven-service-discovery/pkg/writer"
	"github.com/alphagov/paas-prometheus-endpoints/pkg/authenticator"
	"github.com/alphagov/paas-prometheus-endpoints/pkg/metric_endpoint"

	"code.cloudfoundry.org/lager"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/gin-gonic/gin"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
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
	logger := f.logger.Session("fetch-metrics", lager.Data{
		"username": user.Username(),
	})

	prometheusTargets := f.findPrometheusEndpoints(serviceInstances, logger)

	metrics := []metric_endpoint.Metric{}
	for _, prometheusTarget := range prometheusTargets {
		nodeMetricFamilies := f.scrapePrometheusTarget(prometheusTarget, logger)
		targetMetrics := metricsFromNodeMetricFamilies(nodeMetricFamilies, prometheusTarget)
		metrics = append(metrics, targetMetrics...)
	}

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

// FIXME: Should this acceptheader be simplified in case endpoint behaviour will change?
const acceptHeader = `application/openmetrics-text; version=0.0.1,text/plain;version=0.0.4;q=0.5,*/*;q=0.1`
const userAgent = `paas-prometheus-endpoints-elasticsearch`

func (f *ElasticsearchMetricFetcher) scrapePrometheusTarget(target ServiceInstancePrometheusTargetConfig, logger lager.Logger) []map[string]*dto.MetricFamily {
	nodeMetricFamilies := []map[string]*dto.MetricFamily{}
	for _, nodeIP := range target.PrometheusTargetConfig.Targets {
		// Adapted from https://github.com/prometheus/prometheus/blob/f482c7bdd734ba012f6cedd68ddc24b1416efb35/scrape/scrape.go#L601
		// FIXME: Package this as a separate library under the same license,
		// see https://twitter.com/markspolakovs/status/1288953990533853189
		url := fmt.Sprintf("https://%s/metrics", nodeIP.String())
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			logger.Error("error-starting-request-to-target-node", err, lager.Data{
				"prometheus-target": target,
				"node-ip":           nodeIP,
			})
			// FIXME: Log a metric so we know if tenant metrics are broken
			continue
		}
		req.Header.Add("Accept", acceptHeader)
		req.Header.Set("User-Agent", userAgent)
		req.SetBasicAuth(f.aivenPrometheusUsername, f.aivenPrometheusPassword)

		// FIXME: Get a context object in here?
		httpClient := &http.Client{Timeout: 5 * time.Second}
		resp, err := httpClient.Do(req)
		defer resp.Body.Close()
		if err != nil {
			logger.Error("error-making-request-to-target-node", err, lager.Data{
				"prometheus-target": target,
				"node-ip":           nodeIP,
			})
			// FIXME: Log a metric so we know if tenant metrics are broken
			continue
		}

		if resp.StatusCode != http.StatusOK {
			logger.Error(
				"error-non-200-status-code-from-request-to-target-node",
				fmt.Errorf("server returned HTTP status %s", resp.Status),
				lager.Data{
					"prometheus-target": target,
					"node-ip":           nodeIP,
					"status-code":       resp.StatusCode,
				},
			)
			// FIXME: Log a metric so we know if tenant metrics are broken
			continue
		}

		textParser := expfmt.TextParser{}
		metricFamilies, err := textParser.TextToMetricFamilies(resp.Body)
		if err != nil {
			logger.Error("error-parsing-response-from-target-node", err, lager.Data{
				"prometheus-target": target,
				"node-ip":           nodeIP,
			})
			// FIXME: Log a metric so we know if tenant metrics are broken
			continue
		}
		nodeMetricFamilies = append(nodeMetricFamilies, metricFamilies)
	}
	return nodeMetricFamilies
}

func metricsFromNodeMetricFamilies(
	nodeMetricFamilies []map[string]*dto.MetricFamily,
	prometheusTarget ServiceInstancePrometheusTargetConfig,
) []metric_endpoint.Metric {
	metrics := []metric_endpoint.Metric{}
	for i, nodeMetricFamily := range nodeMetricFamilies {
		nodeIP := prometheusTarget.PrometheusTargetConfig.Targets[i]
		for _, metricFamily := range nodeMetricFamily {
			for _, metric := range metricFamily.Metric {
				// FIXME: Add support for non-gauges
				if metric.Gauge == nil {
					continue
				}

				translatedMetric := metric_endpoint.Metric{
					Name:  *metricFamily.Name,
					Value: *metric.Gauge.Value,
					Tags: map[string]string{
						"node-ip":               nodeIP.String(),
						"service_instance_name": prometheusTarget.ServiceInstance.Name,
						"service_instance_guid": prometheusTarget.ServiceInstance.Guid,
						"space_guid":            prometheusTarget.ServiceInstance.SpaceGuid,
						"service_plan_guid":     prometheusTarget.ServiceInstance.ServicePlanGuid,
					},
				}
				for _, labelPair := range metric.Label {
					translatedMetric.Tags[*labelPair.Name] = *labelPair.Value
				}
				metrics = append(metrics, translatedMetric)
			}
		}
	}
	return metrics
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
