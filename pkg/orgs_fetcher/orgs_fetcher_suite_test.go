package orgs_fetcher_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestOrgsFetcher(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OrgsFetcher Suite")
}
