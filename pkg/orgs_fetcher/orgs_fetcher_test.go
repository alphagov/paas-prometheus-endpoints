package orgs_fetcher_test

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/alphagov/paas-prometheus-endpoints/pkg/orgs_fetcher"
	"github.com/alphagov/paas-prometheus-endpoints/pkg/testsupport"

	"code.cloudfoundry.org/lager"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/jarcoal/httpmock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("OrgsFetcher", func() {
	var orgsFetcher *orgs_fetcher.OrgsFetcher
	var fetchSchedule time.Duration

	BeforeEach(func() {
		httpmock.Reset()
		httpclient := &http.Client{Transport: &http.Transport{}}
		httpmock.ActivateNonDefault(httpclient)
		testsupport.SetupCfV2InfoHttpmock()
		testsupport.SetupSuccessfulUaaOauthLoginHttpmock()

		mockCfOrgsApiResponse([]cfclient.Org{
			{
				Guid: "fake-org-1-guid",
				Name: "cf-org-1",
			},
			{
				Guid: "fake-org-2-guid",
				Name: "cf-org-2",
			},
		})

		logger := lager.NewLogger("orgs-fetcher-test")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.INFO))

		cfClient, err := cfclient.NewClient(&cfclient.Config{
			ApiAddress: testsupport.CfApiUrl,
			HttpClient: httpclient,
		})
		Expect(err).NotTo(HaveOccurred())

		fetchSchedule = 400 * time.Millisecond
		orgsFetcher = orgs_fetcher.NewOrgsFetcher(
			fetchSchedule,
			logger,
			cfClient,
		)
	})

	It("provides org metadata from the Cloud Controller API", func() {
		ctx, cancel := context.WithTimeout(context.Background(), fetchSchedule*2)
		defer cancel()
		go orgsFetcher.Run(ctx)
		Eventually(orgsFetcher.GetOrgs).Should(ConsistOf(
			MatchFields(IgnoreExtras, Fields{
				"Guid": Equal("fake-org-1-guid"),
				"Name": Equal("cf-org-1"),
			}),
			MatchFields(IgnoreExtras, Fields{
				"Guid": Equal("fake-org-2-guid"),
				"Name": Equal("cf-org-2"),
			}),
		))
	})

	It("periodically updates the orgs from the Cloud Controller API", func() {
		ctx, cancel := context.WithTimeout(context.Background(), fetchSchedule*4)
		defer cancel()
		go orgsFetcher.Run(ctx)
		Eventually(orgsFetcher.GetOrgs).Should(ConsistOf(
			MatchFields(IgnoreExtras, Fields{
				"Guid": Equal("fake-org-1-guid"),
				"Name": Equal("cf-org-1"),
			}),
			MatchFields(IgnoreExtras, Fields{
				"Guid": Equal("fake-org-2-guid"),
				"Name": Equal("cf-org-2"),
			}),
		))

		mockCfOrgsApiResponse([]cfclient.Org{
			{
				Guid: "fake-org-1-guid",
				Name: "cf-org-1",
			},
			{
				Guid: "fake-org-3-guid",
				Name: "cf-org-3",
			},
		})

		Eventually(orgsFetcher.GetOrgs).Should(ConsistOf(
			MatchFields(IgnoreExtras, Fields{
				"Guid": Equal("fake-org-1-guid"),
				"Name": Equal("cf-org-1"),
			}),
			MatchFields(IgnoreExtras, Fields{
				"Guid": Equal("fake-org-3-guid"),
				"Name": Equal("cf-org-3"),
			}),
		))
		ctx.Done()
	})
})

func wrapOrgForResponse(org cfclient.Org) cfclient.OrgResponse {
	meta := cfclient.Meta{
		Guid:      org.Guid,
		CreatedAt: org.CreatedAt,
	}
	org.Guid = ""
	org.CreatedAt = ""
	return cfclient.OrgResponse{
		Pages: 1,
		Resources: []cfclient.OrgResource{
			{
				Meta:   meta,
				Entity: org,
			},
		},
	}
}

func mockCfOrgsApiResponse(orgs []cfclient.Org) {
	mockURL := fmt.Sprintf("%s/v2/organizations", testsupport.CfApiUrl)
	resp := httpmock.NewJsonResponderOrPanic(
		200, wrapOrgsForResponse(orgs),
	)
	httpmock.RegisterResponderWithQuery("GET", mockURL, nil, resp)
}

func wrapOrgsForResponse(orgs []cfclient.Org) cfclient.OrgResponse {
	orgResources := []cfclient.OrgResource{}
	for _, org := range orgs {
		meta := cfclient.Meta{
			Guid:      org.Guid,
			CreatedAt: org.CreatedAt,
		}
		org.Guid = ""
		org.CreatedAt = ""
		orgResource := cfclient.OrgResource{
			Meta:   meta,
			Entity: org,
		}
		orgResources = append(orgResources, orgResource)
	}
	return cfclient.OrgResponse{
		Pages:     1,
		Resources: orgResources,
	}
}
