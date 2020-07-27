package main

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

func GetMetricsForRedisNodes(
	redisNodes map[string]RedisNode,
	startTime,
	endTime time.Time,
	cloudwatchClient *cloudwatch.CloudWatch,
) (map[string]map[string]*cloudwatch.MetricDataResult, error) {
	nodeMetricQueries := listMetricsForRedisNodes(redisNodes)

	desiredCloudwatchStats := []string{"Average"}
	metricDataQueries, metricDataQueryIdLookup := mergeRedisNodeMetricQueries(nodeMetricQueries, desiredCloudwatchStats)

	metricDataQueriesInGroupsOf500 := batchMetricDataQueriesIntoGroupsOf500(metricDataQueries)

	metricDataResults := []*cloudwatch.MetricDataResult{}
	for _, metricDataQueryInGroupOf500 := range metricDataQueriesInGroupsOf500 {
		pageMetricDataResults, err := fetchUpTo500MetricDataQueries(metricDataQueryInGroupOf500, startTime, endTime, cloudwatchClient)
		if err != nil {
			return nil, err
		}
		metricDataResults = append(metricDataResults, pageMetricDataResults...)
	}

	nodesMetricDataResults := groupMetricDataResultsByNode(metricDataResults, metricDataQueryIdLookup)
	return extractValuesFromMetricDataResults(nodesMetricDataResults, metricDataQueryIdLookup)
}

func listMetricsForRedisNodes(redisNodes map[string]RedisNode) map[string][]*cloudwatch.Metric {
	metricQueries := map[string][]*cloudwatch.Metric{}
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
	cacheClusterId string,
	cacheClusterMetricNames,
	nodeMetricNames []string,
) []*cloudwatch.Metric {
	metricQueries := []*cloudwatch.Metric{}

	for _, metricName := range cacheClusterMetrics {
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
	redisNodeName      string
	metricName         string
	cloudwatchStatName string
}

func mergeRedisNodeMetricQueries(
	nodeMetricQueries map[string][]*cloudwatch.Metric,
	desiredCloudwatchStats []string,
) ([]*cloudwatch.MetricDataQuery, map[string]queryLookup) {
	metricDataQueries := []*cloudwatch.MetricDataQuery{}
	metricDataQueryIdLookup := map[string]queryLookup{}
	metricDataQueryIndex := 0
	for redisNodeName, redisNodeMetricQueries := range nodeMetricQueries {
		for _, redisNodeMetricQuery := range redisNodeMetricQueries {
			for _, desiredCloudwatchStat := range desiredCloudwatchStats {
				metricDataQueryId := fmt.Sprintf("q_%d", metricDataQueryIndex)
				metricDataQuery := &cloudwatch.MetricDataQuery{
					Id: aws.String(metricDataQueryId),
					MetricStat: &cloudwatch.MetricStat{
						Metric: redisNodeMetricQuery,
						Period: aws.Int64(60),
						Stat:   aws.String(desiredCloudwatchStat),
					},
				}
				metricDataQueryIdLookup[metricDataQueryId] = queryLookup{
					redisNodeName:      redisNodeName,
					metricName:         *redisNodeMetricQuery.MetricName,
					cloudwatchStatName: desiredCloudwatchStat,
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
) ([]*cloudwatch.MetricDataResult, error) {
	getMetricDataInput := &cloudwatch.GetMetricDataInput{
		StartTime:         aws.Time(startTime),
		EndTime:           aws.Time(endTime),
		MetricDataQueries: metricDataQueries,
	}

	// FIXME: Remove. Make a proper logger call.
	fmt.Printf("running a getmetricdata api call with %d queries\n", len(getMetricDataInput.MetricDataQueries))
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
) map[string][]*cloudwatch.MetricDataResult {
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
	nodesMetricDataResults map[string][]*cloudwatch.MetricDataResult,
	metricDataQueryIdLookup map[string]queryLookup,
) (map[string]map[string]*cloudwatch.MetricDataResult, error) {
	nodesMetricValues := map[string]map[string]*cloudwatch.MetricDataResult{}
	for nodeName, nodeMetricDataResults := range nodesMetricDataResults {
		nodeMetricValues := map[string]*cloudwatch.MetricDataResult{}

		for _, metricDataResult := range nodeMetricDataResults {
			metricId := *metricDataResult.Id
			metricName := metricDataQueryIdLookup[metricId].metricName
			cloudwatchStatName := metricDataQueryIdLookup[metricId].cloudwatchStatName
			metricKey := fmt.Sprintf(
				"%s_%s",
				pascalCaseToSnakeCase(metricName),
				pascalCaseToSnakeCase(cloudwatchStatName),
			)
			nodeMetricValues[metricKey] = metricDataResult
		}

		nodesMetricValues[nodeName] = nodeMetricValues
	}
	return nodesMetricValues, nil
}

func pascalCaseToSnakeCase(pascalCase string) string {
	// Thanks to https://groups.google.com/g/golang-nuts/c/VCvbLMDE2F0?pli=1
	// FIXME: This changes "CPU" to "c_p_u". Add support for acronyms.
	words := []string{}
	l := 0
	for s := pascalCase; s != ""; s = s[l:] {
		l = strings.IndexFunc(s[1:], unicode.IsUpper) + 1
		if l <= 0 {
			l = len(s)
		}
		words = append(words, s[:l])
	}
	return strings.ToLower(strings.Join(words, "_"))
}

var cacheClusterMetrics = []string{
	"CurrConnections",
	"CurrItems",
	"DatabaseMemoryUsagePercentage",
	"EngineCPUUtilization",
	"NewConnections",
}

var hostMetrics = []string{
	"CPUUtilization",
	"SwapUsage",
}
