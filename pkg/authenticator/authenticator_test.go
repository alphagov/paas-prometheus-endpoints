package authenticator_test

import (
	"fmt"
	"net/http"
	// "net/url"

	"github.com/alphagov/paas-prometheus-endpoints/pkg/authenticator"

	"code.cloudfoundry.org/lager"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/jarcoal/httpmock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Authenticator", func() {
	var serviceInstancePages [][]cfclient.ServiceInstance
	var basicCFUser *authenticator.BasicCFUser

	BeforeEach(func() {
		httpclient := &http.Client{Transport: &http.Transport{}}
		httpmock.ActivateNonDefault(httpclient)

		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("%s/v2/info", cfAPIURL),
			httpmock.NewJsonResponderOrPanic(200, map[string]interface{}{
				"token_endpoint": fmt.Sprintf("%s", uaaAPIURL),
			}),
		)

		cfClient, err := cfclient.NewClient(&cfclient.Config{
			ApiAddress: cfAPIURL,
			HttpClient: httpclient,
		})
		Expect(err).NotTo(HaveOccurred())

		httpmock.Reset() // Reset mock after client creation to clear call count

		logger := lager.NewLogger("cf-user-test")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.INFO))

		basicCFUser = &authenticator.BasicCFUser{CFClient: cfClient}

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

	Context("BasicAuthenticator", func() {
		It("tries to log in with the provided username and password", func() {
			basicAuthenticator := NewBasicAuthenticator(cfAPIURL)
			basicAuthenticator.Authenticate("user", "pass")

			serviceInstances, err := basicCFUser.ListServiceInstancesMatchingPlanGUIDs([]string{"one", "two", "three"})
			Expect(err).ToNot(HaveOccurred())
			Expect(serviceInstances).To(HaveLen(5))
			Eventually(httpmock.GetTotalCallCount).Should(Equal(1))
		})
	})
})

// func mockServiceInstancePageResponse(
// 	page int, totalPages int, addNextURL bool,
// 	expectedQ string,
// 	serviceInstances []cfclient.ServiceInstance,
// ) {
// 	var nextURL string
// 	mockURL := fmt.Sprintf("%s/v2/service_instances", cfAPIURL)

// 	expectedQuery := url.Values{
// 		"q": []string{expectedQ},
// 	}

// 	if page > 1 {
// 		expectedQuery["page"] = []string{fmt.Sprintf("%d", page)}
// 	}

// 	if addNextURL {
// 		nextURLQuery := url.Values{
// 			"q": []string{expectedQ},
// 		}

// 		nextURLQuery["page"] = []string{fmt.Sprintf("%d", page+1)}

// 		nextURL = fmt.Sprintf(
// 			"/v2/service_instances?%s", nextURLQuery.Encode(),
// 		)
// 	}

// 	resp := httpmock.NewJsonResponderOrPanic(
// 		200, wrapServiceInstancesForResponse(totalPages, nextURL, serviceInstances),
// 	)
// 	httpmock.RegisterResponderWithQuery("GET", mockURL, expectedQuery, resp)
// }
