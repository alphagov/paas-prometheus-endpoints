package main

import (
	"fmt"
	"sync"
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
	metricsFrom := time.Now().Add(-10 * time.Minute)
	metricTo := time.Now().Add(-5 * time.Minute)

	redisNodes, err := ListRedisNodes(serviceInstances, f.elasticacheClient)
	if err != nil {
		f.logger.Error("error listing redis nodes", err)
		return nil, err
	}

	// FIXME: This can surely be done with just a channel
	var wg sync.WaitGroup
	bufferedMetrics := make(chan []metric_endpoint.Metric, len(redisNodes))
	for _, redisNode := range redisNodes {
		wg.Add(1)
		go func(redisNode RedisNode) {
			redisNodeMetrics, err := GetRedisNodeMetrics(&redisNode, metricsFrom, metricTo, f.cloudwatchClient)
			if err != nil {
				f.logger.Error("error getting redis metrics for node", err, lager.Data{"node": redisNode.CacheClusterName})
				wg.Done()
				return
			}

			bufferedMetrics <- exportRedisNodeMetrics(redisNodeMetrics, redisNode, &metricTo)
			wg.Done()
		}(redisNode)
	}
	wg.Wait()
	close(bufferedMetrics)

	exportedMetrics := []metric_endpoint.Metric{}
	for bufferMetrics := range bufferedMetrics {
		exportedMetrics = append(exportedMetrics, bufferMetrics...)
	}
	return exportedMetrics, nil
}

func exportRedisNodeMetrics(metrics map[string]interface{}, node RedisNode, when *time.Time) []metric_endpoint.Metric {
	exportedMetrics := []metric_endpoint.Metric{}
	for name, value := range metrics {
		exportedMetric := metric_endpoint.Metric{
			Name:  name,
			Value: value,
			Time:  when,
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
	return exportedMetrics
}
