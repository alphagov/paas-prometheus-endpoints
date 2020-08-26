package main

import (
	"fmt"
	"math"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

type NodeName = string
type MetricName = string

func GetMetricsForRedisNodes(
	redisNodes map[string]RedisNode,
	startTime,
	endTime time.Time,
	cloudwatchClient *cloudwatch.CloudWatch,
	logger lager.Logger,
) (map[NodeName]map[MetricName]*cloudwatch.MetricDataResult, error) {
	logger = logger.Session("get-metrics-for-redis-nodes", lager.Data{
		"number-of-redis-nodes": len(redisNodes),
		"start-time":            startTime.String(),
		"end-time":              endTime.String(),
	})
	nodeMetricQueries := listMetricsForRedisNodes(redisNodes)

	timePeriod := endTime.Sub(startTime)
	timePeriodInSeconds := int64(math.Round(timePeriod.Seconds()))
	metricDataQueries, metricDataQueryIdLookup := createMetricDataQueries(nodeMetricQueries, timePeriodInSeconds)

	metricDataQueriesInGroupsOf500 := batchMetricDataQueriesIntoGroupsOf500(metricDataQueries)

	metricDataResults := []*cloudwatch.MetricDataResult{}
	for _, metricDataQueryInGroupOf500 := range metricDataQueriesInGroupsOf500 {
		pageMetricDataResults, err := fetchUpTo500MetricDataQueries(metricDataQueryInGroupOf500, startTime, endTime, cloudwatchClient, logger)
		if err != nil {
			return nil, err
		}
		metricDataResults = append(metricDataResults, pageMetricDataResults...)
	}

	nodesMetricDataResults := groupMetricDataResultsByNode(metricDataResults, metricDataQueryIdLookup)
	return extractValuesFromMetricDataResults(nodesMetricDataResults, metricDataQueryIdLookup)
}

func listMetricsForRedisNodes(
	redisNodes map[NodeName]RedisNode,
) map[NodeName][]*cloudwatch.Metric {
	metricQueries := map[NodeName][]*cloudwatch.Metric{}
	for _, redisNode := range redisNodes {
		metricQueries[redisNode.CacheClusterName] = listMetricsForRedisNode(
			redisNode.CacheClusterName,
			cacheClusterMetrics,
			hostMetrics,
		)
	}
	return metricQueries
}

func listMetricsForRedisNode(
	cacheClusterId NodeName,
	cacheClusterMetricNames,
	nodeMetricNames []MetricName,
) []*cloudwatch.Metric {
	metricQueries := []*cloudwatch.Metric{}

	for _, metricName := range cacheClusterMetricNames {
		metricQuery := &cloudwatch.Metric{
			Namespace:  aws.String("AWS/ElastiCache"),
			MetricName: aws.String(metricName),
			Dimensions: []*cloudwatch.Dimension{
				{
					Name:  aws.String("CacheClusterId"),
					Value: aws.String(cacheClusterId),
				},
			},
		}
		metricQueries = append(metricQueries, metricQuery)
	}

	for _, metricName := range nodeMetricNames {
		metricQuery := &cloudwatch.Metric{
			Namespace:  aws.String("AWS/ElastiCache"),
			MetricName: aws.String(metricName),
			Dimensions: []*cloudwatch.Dimension{
				{
					Name:  aws.String("CacheClusterId"),
					Value: aws.String(cacheClusterId),
				},
				{
					Name:  aws.String("CacheNodeId"),
					Value: aws.String("0001"),
				},
			},
		}
		metricQueries = append(metricQueries, metricQuery)
	}

	return metricQueries
}

type queryLookup struct {
	redisNodeName string
	metricName    string
	statisticName string
}

func createMetricDataQueries(
	nodeMetricQueries map[NodeName][]*cloudwatch.Metric,
	timePeriodInSeconds int64,
) ([]*cloudwatch.MetricDataQuery, map[string]queryLookup) {
	metricDataQueries := []*cloudwatch.MetricDataQuery{}
	metricDataQueryIdLookup := map[string]queryLookup{}
	metricDataQueryIndex := 0
	for redisNodeName, redisNodeMetricQueries := range nodeMetricQueries {
		for _, redisNodeMetricQuery := range redisNodeMetricQueries {
			for statistic, _ := range statistics {
				metricDataQueryId := fmt.Sprintf("q_%d", metricDataQueryIndex)
				metricDataQuery := &cloudwatch.MetricDataQuery{
					Id: aws.String(metricDataQueryId),
					MetricStat: &cloudwatch.MetricStat{
						Metric: redisNodeMetricQuery,
						Period: aws.Int64(timePeriodInSeconds),
						Stat:   aws.String(statistic),
					},
				}
				metricDataQueryIdLookup[metricDataQueryId] = queryLookup{
					redisNodeName: redisNodeName,
					metricName:    *redisNodeMetricQuery.MetricName,
					statisticName: statistic,
				}
				metricDataQueries = append(metricDataQueries, metricDataQuery)
				metricDataQueryIndex += 1
			}
		}
	}

	return metricDataQueries, metricDataQueryIdLookup
}

func batchMetricDataQueriesIntoGroupsOf500(metricDataQueries []*cloudwatch.MetricDataQuery) [][]*cloudwatch.MetricDataQuery {
	batches := [][]*cloudwatch.MetricDataQuery{}
	for i, metricDataQuery := range metricDataQueries {
		if i%500 == 0 {
			batches = append(batches, []*cloudwatch.MetricDataQuery{})
		}
		batches[len(batches)-1] = append(batches[len(batches)-1], metricDataQuery)
	}
	return batches
}

func fetchUpTo500MetricDataQueries(
	metricDataQueries []*cloudwatch.MetricDataQuery,
	startTime time.Time,
	endTime time.Time,
	cloudwatchClient *cloudwatch.CloudWatch,
	logger lager.Logger,
) ([]*cloudwatch.MetricDataResult, error) {
	getMetricDataInput := &cloudwatch.GetMetricDataInput{
		StartTime:         aws.Time(startTime),
		EndTime:           aws.Time(endTime),
		MetricDataQueries: metricDataQueries,
	}
	logger.Info("get-metric-data-aws-api-call", lager.Data{
		"number-of-queries": len(getMetricDataInput.MetricDataQueries),
	})
	if len(getMetricDataInput.MetricDataQueries) > 500 {
		return nil, fmt.Errorf("more than 500 metric data queries: %d", len(getMetricDataInput.MetricDataQueries))
	}

	getMetricDataOutput, err := cloudwatchClient.GetMetricData(getMetricDataInput)
	if err != nil {
		return nil, fmt.Errorf("error fetching metrics data: %v", err)
	}
	if len(getMetricDataOutput.Messages) > 0 {
		return nil, fmt.Errorf("unexpected issue fetching metrics data: %v", getMetricDataOutput.Messages)
	}
	// FIXME: Fetch multiple pages, just in case
	if getMetricDataOutput.NextToken != nil {
		return nil, fmt.Errorf("more than one page of metrics data results (expected to be unreachable)")
	}
	return getMetricDataOutput.MetricDataResults, nil
}

func groupMetricDataResultsByNode(
	metricDataResults []*cloudwatch.MetricDataResult,
	metricDataQueryIdLookup map[string]queryLookup,
) map[NodeName][]*cloudwatch.MetricDataResult {
	nodesMetricDataQueries := map[string][]*cloudwatch.MetricDataResult{}
	for _, metricDataResult := range metricDataResults {
		metricId := *metricDataResult.Id
		nodeName := metricDataQueryIdLookup[metricId].redisNodeName
		if _, ok := nodesMetricDataQueries[nodeName]; !ok {
			nodesMetricDataQueries[nodeName] = []*cloudwatch.MetricDataResult{}
		}
		nodesMetricDataQueries[nodeName] = append(nodesMetricDataQueries[nodeName], metricDataResult)
	}
	return nodesMetricDataQueries
}

func extractValuesFromMetricDataResults(
	nodesMetricDataResults map[NodeName][]*cloudwatch.MetricDataResult,
	metricDataQueryIdLookup map[string]queryLookup,
) (map[NodeName]map[MetricName]*cloudwatch.MetricDataResult, error) {
	nodesMetricValues := map[NodeName]map[MetricName]*cloudwatch.MetricDataResult{}
	for nodeName, nodeMetricDataResults := range nodesMetricDataResults {
		nodeMetricValues := map[MetricName]*cloudwatch.MetricDataResult{}

		for _, metricDataResult := range nodeMetricDataResults {
			metadata := metricDataQueryIdLookup[*metricDataResult.Id]
			metricKey := fmt.Sprintf(
				"%s_%s",
				metrics[metadata.metricName],
				statistics[metadata.statisticName],
			)
			nodeMetricValues[metricKey] = metricDataResult
		}

		nodesMetricValues[nodeName] = nodeMetricValues
	}
	return nodesMetricValues, nil
}
