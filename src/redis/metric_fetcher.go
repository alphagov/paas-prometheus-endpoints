package main

import (
	"fmt"
	"time"

	"github.com/alphagov/paas-prometheus-endpoints/pkg/authenticator"
	"github.com/alphagov/paas-prometheus-endpoints/pkg/metric_endpoint"

	"code.cloudfoundry.org/lager"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/elasticache"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/gin-gonic/gin"
	dto "github.com/prometheus/client_model/go"
)

type RedisMetricFetcher struct {
	elasticacheClient *elasticache.ElastiCache
	cloudwatchClient  *cloudwatch.CloudWatch
	logger            lager.Logger
}

func NewRedisMetricFetcher(
	elasticacheClient *elasticache.ElastiCache,
	cloudwatchClient *cloudwatch.CloudWatch,
	logger lager.Logger,
) *RedisMetricFetcher {
	logger = logger.Session("redis-metric-fetcher")
	return &RedisMetricFetcher{elasticacheClient, cloudwatchClient, logger}
}

func (f *RedisMetricFetcher) FetchMetrics(
	c *gin.Context,
	user authenticator.User,
	serviceInstances []cfclient.ServiceInstance,
	servicePlans []cfclient.ServicePlan,
	service cfclient.Service,
) (metric_endpoint.Metrics, error) {
	logger := f.logger.Session("fetch-metrics", lager.Data{
		"username": user.Username(),
	})

	redisNodes, err := ListRedisNodes(serviceInstances, f.elasticacheClient)
	if err != nil {
		logger.Error("error listing redis nodes", err)
		return nil, err
	}

	startTime := time.Now().Add(-7 * time.Minute)
	endTime := time.Now().Add(-2 * time.Minute)
	metricDataResults, err := GetMetricsForRedisNodes(redisNodes, startTime, endTime, f.cloudwatchClient, logger)
	if err != nil {
		return nil, err
	}

	promMetrics := metricsFromCloudWatchToPrometheus(metricDataResults, redisNodes, logger)
	return promMetrics, nil
}

func metricsFromCloudWatchToPrometheus(
	metrics map[string]map[string]*cloudwatch.MetricDataResult,
	nodes map[string]RedisNode,
	logger lager.Logger,
) metric_endpoint.Metrics {
	promMetrics := metric_endpoint.Metrics{}
	for nodeName, nodeMetrics := range metrics {
		node := nodes[nodeName]
		for metricName, metricDataResult := range nodeMetrics {
			if _, ok := promMetrics[metricName]; !ok {
				promMetrics[metricName] = &dto.MetricFamily{
					Name:   derefS(metricName),
					Type:   derefT(dto.MetricType_GAUGE),
					Metric: []*dto.Metric{},
				}
			}

			if len(metricDataResult.Timestamps) == 0 || len(metricDataResult.Values) == 0 {
				logger.Error("missing-metric-value", nil, lager.Data{
					"node-name":             node.CacheClusterName,
					"service-instance-guid": node.ServiceInstance.Guid,
					"metric-name":           metricName,
				})
				continue
			}

			timestampMilliseconds := metricDataResult.Timestamps[0].Unix() * 1000
			promMetric := &dto.Metric{
				Label: []*dto.LabelPair{
					{
						Name:  derefS("service_instance_name"),
						Value: derefS(node.ServiceInstance.Name),
					},
					{
						Name:  derefS("service_instance_guid"),
						Value: derefS(node.ServiceInstance.Guid),
					},
					{
						Name:  derefS("space_guid"),
						Value: derefS(node.ServiceInstance.SpaceGuid),
					},
					{
						Name:  derefS("service_plan_guid"),
						Value: derefS(node.ServiceInstance.ServicePlanGuid),
					},
				},
				Gauge: &dto.Gauge{
					Value: metricDataResult.Values[0],
				},
				TimestampMs: &timestampMilliseconds,
			}
			if node.NodeNumber != nil {
				nodeNumber := fmt.Sprintf("%d", *node.NodeNumber)
				promMetric.Label = append(promMetric.Label, &dto.LabelPair{
					Name:  derefS("node"),
					Value: &nodeNumber,
				})
			}
			promMetrics[metricName].Metric = append(promMetrics[metricName].Metric, promMetric)
		}
	}
	return promMetrics
}

func derefS(s string) *string {
	return &s
}

func derefT(i dto.MetricType) *dto.MetricType {
	return &i
}
