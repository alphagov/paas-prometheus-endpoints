package main

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

func GetRedisNodeMetrics(redisNode *RedisNode, startTime, endTime time.Time, cloudwatchClient *cloudwatch.CloudWatch) (map[string]interface{}, error) {
	getMetricDataInput := &cloudwatch.GetMetricDataInput{
		StartTime:         aws.Time(startTime),
		EndTime:           aws.Time(endTime),
		MetricDataQueries: buildMetricDataQueries(redisNode.CacheClusterName, cacheClusterMetrics, hostMetrics),
	}

	getMetricDataOutput, err := cloudwatchClient.GetMetricData(getMetricDataInput)
	if err != nil {
		return nil, fmt.Errorf("error fetching metrics data: %v", err)
	}
	if len(getMetricDataOutput.Messages) > 0 {
		return nil, fmt.Errorf("unexpected issue fetching metrics data: %v", getMetricDataOutput.Messages)
	}
	if getMetricDataOutput.NextToken != nil {
		return nil, fmt.Errorf("more than one page of metrics data results (expected to be unreachable)")
	}

	metrics := map[string]interface{}{}
	for _, metricDataResult := range getMetricDataOutput.MetricDataResults {
		metricName := *metricDataResult.Id
		if metricDataResult.Values == nil {
			metrics[metricName] = nil
		} else if len(metricDataResult.Values) == 1 {
			metrics[metricName] = *metricDataResult.Values[0]
		} else {
			return nil, fmt.Errorf("metric data result had more than one value (expected to be unreachable)")
		}
	}
	return metrics, nil
}

func buildMetricDataQueries(cacheClusterId string, cacheClusterMetricNames, nodeMetricNames []string) []*cloudwatch.MetricDataQuery {
	metricDataQueries := []*cloudwatch.MetricDataQuery{}

	for _, metricName := range cacheClusterMetrics {
		metricDataQuery := &cloudwatch.MetricDataQuery{
			Id: aws.String(pascalCaseToSnakeCase(metricName)),
			MetricStat: &cloudwatch.MetricStat{
				Metric: &cloudwatch.Metric{
					Namespace:  aws.String("AWS/ElastiCache"),
					MetricName: aws.String(metricName),
					Dimensions: []*cloudwatch.Dimension{
						{
							Name:  aws.String("CacheClusterId"),
							Value: aws.String(cacheClusterId),
						},
					},
				},
				// FIXME: Relate this to the start/end time of the GetMetricData
				Period: aws.Int64(300),
				// FIXME: Explore alternative Stats
				Stat: aws.String("Average"),
			},
		}
		metricDataQueries = append(metricDataQueries, metricDataQuery)
	}

	for _, metricName := range nodeMetricNames {
		metricDataQuery := &cloudwatch.MetricDataQuery{
			Id: aws.String(pascalCaseToSnakeCase(metricName)),
			MetricStat: &cloudwatch.MetricStat{
				Metric: &cloudwatch.Metric{
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
				},
				// FIXME: Relate this to the start/end time of the GetMetricData
				Period: aws.Int64(300),
				// FIXME: Explore alternative Stats
				Stat: aws.String("Average"),
			},
		}
		metricDataQueries = append(metricDataQueries, metricDataQuery)
	}

	return metricDataQueries
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
	"ActiveDefragHits",
	"BytesUsedForCache",
	"CacheHits",
	"CacheMisses",
	"CacheHitRate",
	"CurrConnections",
	"DatabaseMemoryUsagePercentage",
	"DB0AverageTTL",
	"EngineCPUUtilization",
	"Evictions",
	"MasterLinkHealthStatus",
	"MemoryFragmentationRatio",
	"NewConnections",
	"Reclaimed",
	"ReplicationBytes",
	"ReplicationLag",
	"SaveInProgress",
	"CurrItems",
	"EvalBasedCmds",
	"EvalBasedCmdsLatency",
	"GeoSpatialBasedCmds",
	"GeoSpatialBasedCmdsLatency",
	"GetTypeCmds",
	"GetTypeCmdsLatency",
	"HashBasedCmds",
	"HashBasedCmdsLatency",
	"HyperLogLogBasedCmds",
	"HyperLogLogBasedCmdsLatency",
	"KeyBasedCmds",
	"KeyBasedCmdsLatency",
	"ListBasedCmds",
	"ListBasedCmdsLatency",
	"PubSubBasedCmds",
	"PubSubBasedCmdsLatency",
	"SetBasedCmds",
	"SetBasedCmdsLatency",
	"SetTypeCmds",
	"SetTypeCmdsLatency",
	"SortedSetBasedCmds",
	"SortedBasedCmdsLatency",
	"StringBasedCmds",
	"StringBasedCmdsLatency",
	"StreamBasedCmds",
	"StreamBasedCmdsLatency",
}

var hostMetrics = []string{
	"CPUUtilization",
	"FreeableMemory",
	"NetworkBytesIn",
	"NetworkBytesOut",
	"NetworkPacketsIn",
	"NetworkPacketsOut",
	"SwapUsage",
}
