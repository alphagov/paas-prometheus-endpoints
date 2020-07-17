package authenticator

import (
	"fmt"
	"net/http"
	"os"
	"time"

	cfclient "github.com/cloudfoundry-community/go-cfclient"
)

type Authenticator interface {
	Authenticate(username, password string) (CFUser, error)
}

type BasicAuthenticator struct {
	cfURL string
}

func NewBasicAuthenticator(cfURL string) *BasicAuthenticator {
	return &BasicAuthenticator{cfURL}
}

func (a *BasicAuthenticator) Authenticate(username, password string) (CFUser, error) {
	CFClient, err := cfclient.NewClient(&cfclient.Config{
		ApiAddress:        a.cfURL,
		Username:          username,
		Password:          password,
		SkipSslValidation: os.Getenv("CF_SKIP_SSL_VALIDATION") == "true",
		UserAgent:         os.Getenv("CF_USER_AGENT"),
		HttpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error authenticating user: %v", err)
	}
	return BasicCFUser{CFClient}, nil
}
