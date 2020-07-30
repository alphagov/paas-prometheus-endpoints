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
	user authenticator.CFUser,
	serviceInstances []cfclient.ServiceInstance,
	servicePlans []cfclient.ServicePlan,
	service cfclient.Service,
) ([]metric_endpoint.Metric, error) {
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

	metrics := []metric_endpoint.Metric{}
	for redisNodeName, nodeMetricDataResults := range metricDataResults {
		redisNode := redisNodes[redisNodeName]
		nodeMetrics := exportRedisNodeMetrics(nodeMetricDataResults, redisNode)
		metrics = append(metrics, nodeMetrics...)
	}

	return metrics, nil
}

func exportRedisNodeMetrics(metrics map[string]*cloudwatch.MetricDataResult, node RedisNode) []metric_endpoint.Metric {
	exportedMetrics := []metric_endpoint.Metric{}
	for metricKey, metricDataResult := range metrics {
		for i := len(metricDataResult.Values) - 1; i >= 0; i-- {
			timestamp := *metricDataResult.Timestamps[i]
			value := *metricDataResult.Values[i]
			exportedMetric := metric_endpoint.Metric{
				Name:  metricKey,
				Value: value,
				Time:  &timestamp,
				Tags: map[string]string{
					"service_instance_name": node.ServiceInstance.Name,
					"service_instance_guid": node.ServiceInstance.Guid,
					"space_guid":            node.ServiceInstance.SpaceGuid,
					"service_plan_guid":     node.ServiceInstance.ServicePlanGuid,
				},
			}
			if node.NodeNumber != nil {
				exportedMetric.Tags["node"] = fmt.Sprintf("%d", *node.NodeNumber)
			}
			exportedMetrics = append(exportedMetrics, exportedMetric)
		}
	}
	return exportedMetrics
}
