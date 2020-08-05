package authenticator_test

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/alphagov/paas-prometheus-endpoints/pkg/authenticator"
	"github.com/alphagov/paas-prometheus-endpoints/pkg/testsupport"

	"code.cloudfoundry.org/lager"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/jarcoal/httpmock"
	"github.com/jinzhu/copier"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("User", func() {
	var serviceInstancePages [][]cfclient.ServiceInstance
	var basicUser *authenticator.BasicUser

	BeforeEach(func() {
		httpclient := &http.Client{Transport: &http.Transport{}}
		httpmock.ActivateNonDefault(httpclient)
		testsupport.SetupCfV2InfoHttpmock()
		testsupport.SetupSuccessfulUaaOauthLoginHttpmock()

		cfClient, err := cfclient.NewClient(&cfclient.Config{
			ApiAddress: testsupport.CfApiUrl,
			HttpClient: httpclient,
		})
		Expect(err).NotTo(HaveOccurred())
		basicUser = authenticator.NewBasicUser(cfClient, "test-username")
		httpmock.Reset() // Reset mock after client creation to clear call count

		logger := lager.NewLogger("user-test")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.INFO))

		serviceInstancePages = [][]cfclient.ServiceInstance{
			{
				{Guid: "a"},
				{Guid: "b"},
				{Guid: "c"},
			},
			{
				{Guid: "d"},
				{Guid: "e"},
			},
		}
	})

	Context("BasicUser", func() {
		It("queries cloud foundry for service instances matching provided service plan guids", func() {
			mockServiceInstancePageResponse(
				1, 2, true,
				"service_plan_guid IN one,two,three",
				serviceInstancePages[0],
			)
			mockServiceInstancePageResponse(
				2, 2, false,
				"service_plan_guid IN one,two,three",
				serviceInstancePages[1],
			)

			serviceInstances, err := basicUser.ListServiceInstancesMatchingPlanGUIDs([]string{"one", "two", "three"})
			Expect(err).ToNot(HaveOccurred())
			Expect(serviceInstances).To(HaveLen(5))
			Eventually(httpmock.GetTotalCallCount).Should(Equal(2))
		})
	})
})

func mockServiceInstancePageResponse(
	page int, totalPages int, addNextURL bool,
	expectedQ string,
	serviceInstances []cfclient.ServiceInstance,
) {
	var nextURL string
	mockURL := fmt.Sprintf("%s/v2/service_instances", testsupport.CfApiUrl)

	expectedQuery := url.Values{
		"q": []string{expectedQ},
	}

	if page > 1 {
		expectedQuery["page"] = []string{fmt.Sprintf("%d", page)}
	}

	if addNextURL {
		nextURLQuery := url.Values{
			"q": []string{expectedQ},
		}

		nextURLQuery["page"] = []string{fmt.Sprintf("%d", page+1)}

		nextURL = fmt.Sprintf(
			"/v2/service_instances?%s", nextURLQuery.Encode(),
		)
	}

	resp := httpmock.NewJsonResponderOrPanic(
		200, wrapServiceInstancesForResponse(totalPages, nextURL, serviceInstances),
	)
	httpmock.RegisterResponderWithQuery("GET", mockURL, expectedQuery, resp)
}

func wrapServiceInstancesForResponse(
	pages int,
	nextURL string,
	serviceInstances []cfclient.ServiceInstance,
) cfclient.ServiceInstancesResponse {
	serviceInstanceResources := make([]cfclient.ServiceInstanceResource, len(serviceInstances))
	for i, serviceInstance := range serviceInstances {
		// We do not want CreatedAt and GUID as they are not in the API response
		var serviceInstanceWithoutFields cfclient.ServiceInstance
		copier.Copy(&serviceInstanceWithoutFields, &serviceInstance)
		serviceInstanceWithoutFields.Guid = ""
		serviceInstanceWithoutFields.CreatedAt = ""

		serviceInstanceResources[i] = cfclient.ServiceInstanceResource{
			Meta: cfclient.Meta{
				Guid:      serviceInstance.Guid,
				CreatedAt: serviceInstance.CreatedAt,
			},
			Entity: serviceInstanceWithoutFields,
		}
	}

	return cfclient.ServiceInstancesResponse{
		Pages:     pages,
		NextUrl:   nextURL,
		Resources: serviceInstanceResources,
	}
}
