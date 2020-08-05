package authenticator_test

import (
	"fmt"
	"net/http"

	"github.com/alphagov/paas-prometheus-endpoints/pkg/authenticator"

	"code.cloudfoundry.org/lager"
	"github.com/jarcoal/httpmock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Authenticator", func() {
	var basicAuthenticator *authenticator.BasicAuthenticator

	BeforeEach(func() {
		httpmock.Reset()
		httpclient := &http.Client{Transport: &http.Transport{}}
		httpmock.ActivateNonDefault(httpclient)
		setupCfV2InfoHttpmock()

		logger := lager.NewLogger("cf-user-test")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.INFO))

		basicAuthenticator = authenticator.NewBasicAuthenticator(cfApiUrl, httpclient)
	})

	Context("BasicAuthenticator", func() {
		It("tries to log in with the provided username and password", func() {
			setupSuccessfulUaaOauthLoginHttpmock()

			user, err := basicAuthenticator.Authenticate("user", "pass")
			Expect(err).ToNot(HaveOccurred())
			Expect(user.Username()).To(Equal("user"))

			httpmockInfo := httpmock.GetCallCountInfo()
			Expect(httpmockInfo[fmt.Sprintf("GET %s/v2/info", cfApiUrl)]).Should(Equal(1))
			Expect(httpmockInfo[fmt.Sprintf("GET %s/v2/info", cfApiUrl)]).Should(Equal(1))
			Expect(httpmockInfo).To(HaveLen(2))
		})

		It("returns an error if UAA does not accept the credentials", func() {
			setupFailedUaaOauthLoginHttpmock()

			user, err := basicAuthenticator.Authenticate("user", "pass")
			Expect(err).To(HaveOccurred())
			Expect(user).To(BeNil())

			httpmockInfo := httpmock.GetCallCountInfo()
			Expect(httpmockInfo[fmt.Sprintf("GET %s/v2/info", cfApiUrl)]).Should(Equal(1))
			Expect(httpmockInfo[fmt.Sprintf("GET %s/v2/info", cfApiUrl)]).Should(Equal(1))
			Expect(httpmockInfo).To(HaveLen(2))
		})
	})
})
