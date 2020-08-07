package metric_endpoint

import (
	"io"

	"github.com/alphagov/paas-prometheus-endpoints/pkg/authenticator"

	"code.cloudfoundry.org/lager"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/gin-gonic/gin"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

type Metrics = map[string]*dto.MetricFamily

type ServiceMetricFetcher interface {
	FetchMetrics(
		c *gin.Context,
		user authenticator.User,
		serviceInstances []cfclient.ServiceInstance,
		servicePlans []cfclient.ServicePlan,
		service cfclient.Service,
	) (Metrics, error)
}

func renderMetrics(metrics Metrics, out io.Writer, logger lager.Logger) int {
	totalBytesWritten := 0
	for _, metricFamily := range metrics {
		bytesWritten, err := expfmt.MetricFamilyToText(out, metricFamily)
		totalBytesWritten += bytesWritten
		if err != nil {
			// FIXME: Report a metric for this so we know if it starts to error?
			logger.Error("error-rendering-metrics", err, lager.Data{
				"metric-family-name": metricFamily.Name,
			})
			continue
		}
	}
	return totalBytesWritten
}
