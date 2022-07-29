package spaces_fetcher_test

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/alphagov/paas-prometheus-endpoints/pkg/spaces_fetcher"
	"github.com/alphagov/paas-prometheus-endpoints/pkg/testsupport"

	"code.cloudfoundry.org/lager"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/jarcoal/httpmock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("SpacesFetcher", func() {
	var spacesFetcher *spaces_fetcher.SpacesFetcher
	var fetchSchedule time.Duration

	BeforeEach(func() {
		httpmock.Reset()
		httpclient := &http.Client{Transport: &http.Transport{}}
		httpmock.ActivateNonDefault(httpclient)
		testsupport.SetupCfV2InfoHttpmock()
		testsupport.SetupSuccessfulUaaOauthLoginHttpmock()

		mockCfSpacesApiResponse([]cfclient.Space{
			{
				Guid: "fake-space-1-guid",
				Name: "cf-space-1",
			},
			{
				Guid: "fake-space-2-guid",
				Name: "cf-space-2",
			},
		})

		logger := lager.NewLogger("spaces-fetcher-test")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.INFO))

		cfClient, err := cfclient.NewClient(&cfclient.Config{
			ApiAddress: testsupport.CfApiUrl,
			HttpClient: httpclient,
		})
		Expect(err).NotTo(HaveOccurred())

		fetchSchedule = 400 * time.Millisecond
		spacesFetcher = spaces_fetcher.NewSpacesFetcher(
			fetchSchedule,
			logger,
			cfClient,
		)
	})

	It("provides space metadata from the Cloud Controller API", func() {
		ctx, cancel := context.WithTimeout(context.Background(), fetchSchedule*2)
		defer cancel()
		go spacesFetcher.Run(ctx)
		Eventually(spacesFetcher.GetSpaces).Should(ConsistOf(
			MatchFields(IgnoreExtras, Fields{
				"Guid": Equal("fake-space-1-guid"),
				"Name": Equal("cf-space-1"),
			}),
			MatchFields(IgnoreExtras, Fields{
				"Guid": Equal("fake-space-2-guid"),
				"Name": Equal("cf-space-2"),
			}),
		))
	})

	It("periodically updates the spaces from the Cloud Controller API", func() {
		ctx, cancel := context.WithTimeout(context.Background(), fetchSchedule*4)
		defer cancel()
		go spacesFetcher.Run(ctx)
		Eventually(spacesFetcher.GetSpaces).Should(ConsistOf(
			MatchFields(IgnoreExtras, Fields{
				"Guid": Equal("fake-space-1-guid"),
				"Name": Equal("cf-space-1"),
			}),
			MatchFields(IgnoreExtras, Fields{
				"Guid": Equal("fake-space-2-guid"),
				"Name": Equal("cf-space-2"),
			}),
		))

		mockCfSpacesApiResponse([]cfclient.Space{
			{
				Guid: "fake-space-1-guid",
				Name: "cf-space-1",
			},
			{
				Guid: "fake-space-3-guid",
				Name: "cf-space-3",
			},
		})

		Eventually(spacesFetcher.GetSpaces).Should(ConsistOf(
			MatchFields(IgnoreExtras, Fields{
				"Guid": Equal("fake-space-1-guid"),
				"Name": Equal("cf-space-1"),
			}),
			MatchFields(IgnoreExtras, Fields{
				"Guid": Equal("fake-space-3-guid"),
				"Name": Equal("cf-space-3"),
			}),
		))
		ctx.Done()
	})
})

func wrapSpaceForResponse(space cfclient.Space) cfclient.SpaceResponse {
	meta := cfclient.Meta{
		Guid:      space.Guid,
		CreatedAt: space.CreatedAt,
	}
	space.Guid = ""
	space.CreatedAt = ""
	return cfclient.SpaceResponse{
		Pages: 1,
		Resources: []cfclient.SpaceResource{
			{
				Meta:   meta,
				Entity: space,
			},
		},
	}
}

func mockCfSpacesApiResponse(spaces []cfclient.Space) {
	mockURL := fmt.Sprintf("%s/v2/spaces", testsupport.CfApiUrl)
	resp := httpmock.NewJsonResponderOrPanic(
		200, wrapSpacesForResponse(spaces),
	)
	httpmock.RegisterResponderWithQuery("GET", mockURL, nil, resp)
}

func wrapSpacesForResponse(spaces []cfclient.Space) cfclient.SpaceResponse {
	spaceResources := []cfclient.SpaceResource{}
	for _, space := range spaces {
		meta := cfclient.Meta{
			Guid:      space.Guid,
			CreatedAt: space.CreatedAt,
		}
		space.Guid = ""
		space.CreatedAt = ""
		spaceResource := cfclient.SpaceResource{
			Meta:   meta,
			Entity: space,
		}
		spaceResources = append(spaceResources, spaceResource)
	}
	return cfclient.SpaceResponse{
		Pages:     1,
		Resources: spaceResources,
	}
}
