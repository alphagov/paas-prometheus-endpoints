package authenticator_test

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/alphagov/paas-prometheus-endpoints/pkg/authenticator"

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

		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("%s/v2/info", cfApiUrl),
			httpmock.NewJsonResponderOrPanic(200, map[string]interface{}{
				"token_endpoint": fmt.Sprintf("%s", uaaApiUrl),
			}),
		)

		httpmock.RegisterResponder(
			"POST",
			fmt.Sprintf("%s/oauth/token", uaaApiUrl),
			httpmock.NewJsonResponderOrPanic(200, map[string]interface{}{
				// Copy and pasted from UAA docs
				"access_token":  "acb6803a48114d9fb4761e403c17f812",
				"token_type":    "bearer",
				"id_token":      "eyJhbGciOiJIUzI1NiIsImprdSI6Imh0dHBzOi8vbG9jYWxob3N0OjgwODAvdWFhL3Rva2VuX2tleXMiLCJraWQiOiJsZWdhY3ktdG9rZW4ta2V5IiwidHlwIjoiSldUIn0.eyJzdWIiOiIwNzYzZTM2MS02ODUwLTQ3N2ItYjk1Ny1iMmExZjU3MjczMTQiLCJhdWQiOlsibG9naW4iXSwiaXNzIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwL3VhYS9vYXV0aC90b2tlbiIsImV4cCI6MTU1NzgzMDM4NSwiaWF0IjoxNTU3Nzg3MTg1LCJhenAiOiJsb2dpbiIsInNjb3BlIjpbIm9wZW5pZCJdLCJlbWFpbCI6IndyaHBONUB0ZXN0Lm9yZyIsInppZCI6InVhYSIsIm9yaWdpbiI6InVhYSIsImp0aSI6ImFjYjY4MDNhNDgxMTRkOWZiNDc2MWU0MDNjMTdmODEyIiwiZW1haWxfdmVyaWZpZWQiOnRydWUsImNsaWVudF9pZCI6ImxvZ2luIiwiY2lkIjoibG9naW4iLCJncmFudF90eXBlIjoiYXV0aG9yaXphdGlvbl9jb2RlIiwidXNlcl9uYW1lIjoid3JocE41QHRlc3Qub3JnIiwicmV2X3NpZyI6ImI3MjE5ZGYxIiwidXNlcl9pZCI6IjA3NjNlMzYxLTY4NTAtNDc3Yi1iOTU3LWIyYTFmNTcyNzMxNCIsImF1dGhfdGltZSI6MTU1Nzc4NzE4NX0.Fo8wZ_Zq9mwFks3LfXQ1PfJ4ugppjWvioZM6jSqAAQQ",
				"refresh_token": "f59dcb5dcbca45f981f16ce519d61486-r",
				"expires_in":    43199,
				"scope":         "openid oauth.approvals",
				"jti":           "acb6803a48114d9fb4761e403c17f812",
			}),
		)

		cfClient, err := cfclient.NewClient(&cfclient.Config{
			ApiAddress: cfApiUrl,
			HttpClient: httpclient,
		})
		Expect(err).NotTo(HaveOccurred())
		basicUser = authenticator.NewBasicUser(cfClient, "test-username")
		httpmock.Reset() // Reset mock after client creation to clear call count

		logger := lager.NewLogger("cf-user-test")
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
	mockURL := fmt.Sprintf("%s/v2/service_instances", cfApiUrl)

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
