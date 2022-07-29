package spaces_fetcher_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSpacesFetcher(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SpacesFetcher Suite")
}
