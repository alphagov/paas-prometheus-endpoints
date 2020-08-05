package authenticator_test

import (
	"net/http"
	"net/http/httptest"

	a "github.com/alphagov/paas-prometheus-endpoints/pkg/authenticator"

	"code.cloudfoundry.org/lager"
	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AuthenticatorMiddleware", func() {
	var authenticator *a.MockAuthenticator
	var middleware gin.HandlerFunc
	var router *gin.Engine

	BeforeEach(func() {
		logger := lager.NewLogger("authenticator-middleware-test")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.INFO))

		authenticator = &a.MockAuthenticator{
			AllowedUsername: "allowed-username",
			AllowedPassword: "allowed-password",
		}
		middleware = a.AuthenticatorMiddleware(authenticator, logger)

		router = gin.Default()
		router.Use(middleware)
	})

	It("passes the user to the next handler when login is successful", func() {
		router.GET("/protected-endpoint", func(c *gin.Context) {
			user := c.MustGet("authenticated_user").(a.User)
			c.String(http.StatusOK, user.Username())
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/protected-endpoint", nil)
		req.Header.Set("Authorization", authorizationHeader("allowed-username", "allowed-password"))
		router.ServeHTTP(w, req)
		Expect(w.Body.String()).To(Equal("allowed-username"))
		Expect(w.Code).To(Equal(http.StatusOK))
	})

	It("responds with an error when there are no basic auth credentials", func() {
		router.GET("/protected-endpoint", func(c *gin.Context) {
			user := c.MustGet("authenticated_user").(a.User)
			c.String(http.StatusOK, user.Username())
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/protected-endpoint", nil)
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusUnauthorized))
	})

	It("responds with an error when the basic auth credentials are invalid", func() {
		router.GET("/protected-endpoint", func(c *gin.Context) {
			user := c.MustGet("authenticated_user").(a.User)
			c.String(http.StatusOK, user.Username())
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/protected-endpoint", nil)
		req.Header.Set("Authorization", authorizationHeader("allowed-username", "wrong-password"))
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusUnauthorized))
	})
})
