package authenticator

import (
	"net/http"

	"code.cloudfoundry.org/lager"
	"github.com/gin-gonic/gin"
)

func AuthenticatorMiddleware(auth Authenticator, logger lager.Logger) gin.HandlerFunc {
	logger = logger.Session("authenticator-middleware")
	return func(c *gin.Context) {
		username, password, ok := c.Request.BasicAuth()
		if !ok {
			logger.Error("err-request-did-not-provide-credentials", nil)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "you must provide user credentials via http basic auth",
			})
			return
		}

		user, err := auth.Authenticate(username, password)
		if err != nil {
			logger.Error("err-request-credentials-did-not-work", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "provided credentials did not login successfully",
			})
			return
		}

		logger.Info("successfully-authenticated-user", lager.Data{"username": username})
		c.Set("authenticated_user", user)

		c.Next()
	}
}
