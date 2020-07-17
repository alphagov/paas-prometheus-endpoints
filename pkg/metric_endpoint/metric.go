package metric_endpoint

import (
	"fmt"
	"time"

	"github.com/alphagov/paas-prometheus-endpoints/pkg/authenticator"

	"code.cloudfoundry.org/lager"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/gin-gonic/gin"
)

type Metric struct {
	Name  string
	Value interface{}
	Time  *time.Time
	Tags  map[string]string
}

type ServiceMetricFetcher func(
	c *gin.Context,
	user authenticator.CFUser,
	serviceInstances []cfclient.ServiceInstance,
	servicePlans []cfclient.ServicePlan,
	service cfclient.Service,
	logger lager.Logger,
) ([]Metric, error)

func groupMetricsByName(metrics []Metric) map[string][]Metric {
	groupedMetrics := map[string][]Metric{}
	for _, metric := range metrics {
		if _, ok := groupedMetrics[metric.Name]; !ok {
			groupedMetrics[metric.Name] = []Metric{}
		}
		groupedMetrics[metric.Name] = append(groupedMetrics[metric.Name], metric)
	}
	return groupedMetrics
}

func renderMetricGroup(metricName string, metricGroup []Metric) string {
	// FIXME: Support non-gauges metrics?
	output := fmt.Sprintf("# HELP %s\n", metricName)
	output += fmt.Sprintf("# TYPE %s gauge\n", metricName)
	for _, metric := range metricGroup {
		if metric.Value == nil {
			continue
		}

		output += fmt.Sprintf("%s{", metricName)
		firstTag := true
		for tagName, tagValue := range metric.Tags {
			if !firstTag {
				output += ","
			}
			output += fmt.Sprintf("%s=\"%s\"", tagName, tagValue)
			firstTag = false
		}
		// FIXME: Output timestamp too
		output += fmt.Sprintf("} %v\n", metric.Value)
	}
	return output
}
