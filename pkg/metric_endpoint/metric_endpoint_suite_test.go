package metric_endpoint_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMetricEndpoint(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Metric Endpoint Suite")
}
