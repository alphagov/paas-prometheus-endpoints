package service_plans_fetcher_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestServicePlansFetcher(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ServicePlansFetcher Suite")
}
