package service_plans_fetcher_test

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/alphagov/paas-prometheus-endpoints/pkg/service_plans_fetcher"
	"github.com/alphagov/paas-prometheus-endpoints/pkg/testsupport"

	"code.cloudfoundry.org/lager"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/jarcoal/httpmock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("Authenticator", func() {
	var servicePlansFetcher *service_plans_fetcher.ServicePlansFetcher
	var fetchSchedule time.Duration

	BeforeEach(func() {
		httpmock.Reset()
		httpclient := &http.Client{Transport: &http.Transport{}}
		httpmock.ActivateNonDefault(httpclient)
		testsupport.SetupCfV2InfoHttpmock()
		testsupport.SetupSuccessfulUaaOauthLoginHttpmock()

		mockCfServicesApiResponse("label:cf-service-name", cfclient.Service{
			Guid:  "fake-service-guid",
			Label: "cf-service-name",
		})

		mockCfServicePlansApiResponse("service_guid:fake-service-guid", []cfclient.ServicePlan{
			{
				Guid: "fake-service-plan-1-guid",
				Name: "cf-service-plan-1",
			},
			{
				Guid: "fake-service-plan-2-guid",
				Name: "cf-service-plan-2",
			},
		})

		logger := lager.NewLogger("service-plans-fetcher-test")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.INFO))

		cfClient, err := cfclient.NewClient(&cfclient.Config{
			ApiAddress: testsupport.CfApiUrl,
			HttpClient: httpclient,
		})
		Expect(err).NotTo(HaveOccurred())

		fetchSchedule = 400 * time.Millisecond
		servicePlansFetcher = service_plans_fetcher.NewServicePlansFetcher(
			"cf-service-name",
			fetchSchedule,
			logger,
			cfClient,
		)
	})

	It("provides service metadata from the Cloud Controller API", func() {
		ctx, cancel := context.WithTimeout(context.Background(), fetchSchedule*2)
		defer cancel()
		go servicePlansFetcher.Run(ctx)
		Eventually(servicePlansFetcher.GetService).Should(PointTo(MatchFields(IgnoreExtras, Fields{
			"Guid":  Equal("fake-service-guid"),
			"Label": Equal("cf-service-name"),
		})))
	})

	It("provides service plans metadata from the Cloud Controller API", func() {
		ctx, cancel := context.WithTimeout(context.Background(), fetchSchedule*2)
		defer cancel()
		go servicePlansFetcher.Run(ctx)
		Eventually(servicePlansFetcher.GetServicePlans).Should(ConsistOf(
			MatchFields(IgnoreExtras, Fields{
				"Guid": Equal("fake-service-plan-1-guid"),
				"Name": Equal("cf-service-plan-1"),
			}),
			MatchFields(IgnoreExtras, Fields{
				"Guid": Equal("fake-service-plan-2-guid"),
				"Name": Equal("cf-service-plan-2"),
			}),
		))
	})

	It("periodically updates the service plans from the Cloud Controller API", func() {
		ctx, cancel := context.WithTimeout(context.Background(), fetchSchedule*4)
		defer cancel()
		go servicePlansFetcher.Run(ctx)
		Eventually(servicePlansFetcher.GetServicePlans).Should(ConsistOf(
			MatchFields(IgnoreExtras, Fields{
				"Guid": Equal("fake-service-plan-1-guid"),
				"Name": Equal("cf-service-plan-1"),
			}),
			MatchFields(IgnoreExtras, Fields{
				"Guid": Equal("fake-service-plan-2-guid"),
				"Name": Equal("cf-service-plan-2"),
			}),
		))

		mockCfServicePlansApiResponse("service_guid:fake-service-guid", []cfclient.ServicePlan{
			{
				Guid: "fake-service-plan-1-guid",
				Name: "cf-service-plan-1",
			},
			{
				Guid: "fake-service-plan-3-guid",
				Name: "cf-service-plan-3",
			},
		})

		Eventually(servicePlansFetcher.GetServicePlans).Should(ConsistOf(
			MatchFields(IgnoreExtras, Fields{
				"Guid": Equal("fake-service-plan-1-guid"),
				"Name": Equal("cf-service-plan-1"),
			}),
			MatchFields(IgnoreExtras, Fields{
				"Guid": Equal("fake-service-plan-3-guid"),
				"Name": Equal("cf-service-plan-3"),
			}),
		))
		ctx.Done()
	})
})

func mockCfServicesApiResponse(expectedQ string, service cfclient.Service) {
	mockURL := fmt.Sprintf("%s/v2/services", testsupport.CfApiUrl)
	expectedQuery := url.Values{
		"q": []string{expectedQ},
	}
	resp := httpmock.NewJsonResponderOrPanic(
		200, wrapServiceForResponse(service),
	)
	httpmock.RegisterResponderWithQuery("GET", mockURL, expectedQuery, resp)
}

func wrapServiceForResponse(service cfclient.Service) cfclient.ServicesResponse {
	meta := cfclient.Meta{
		Guid:      service.Guid,
		CreatedAt: service.CreatedAt,
	}
	service.Guid = ""
	service.CreatedAt = ""
	return cfclient.ServicesResponse{
		Pages: 1,
		Resources: []cfclient.ServicesResource{
			{
				Meta:   meta,
				Entity: service,
			},
		},
	}
}

func mockCfServicePlansApiResponse(expectedQ string, servicePlans []cfclient.ServicePlan) {
	mockURL := fmt.Sprintf("%s/v2/service_plans", testsupport.CfApiUrl)
	expectedQuery := url.Values{
		"q": []string{expectedQ},
	}
	resp := httpmock.NewJsonResponderOrPanic(
		200, wrapServicePlansForResponse(servicePlans),
	)
	httpmock.RegisterResponderWithQuery("GET", mockURL, expectedQuery, resp)
}

func wrapServicePlansForResponse(servicePlans []cfclient.ServicePlan) cfclient.ServicePlansResponse {
	servicePlanResources := []cfclient.ServicePlanResource{}
	for _, servicePlan := range servicePlans {
		meta := cfclient.Meta{
			Guid:      servicePlan.Guid,
			CreatedAt: servicePlan.CreatedAt,
		}
		servicePlan.Guid = ""
		servicePlan.CreatedAt = ""
		servicePlanResource := cfclient.ServicePlanResource{
			Meta:   meta,
			Entity: servicePlan,
		}
		servicePlanResources = append(servicePlanResources, servicePlanResource)
	}
	return cfclient.ServicePlansResponse{
		Pages:     1,
		Resources: servicePlanResources,
	}
}
