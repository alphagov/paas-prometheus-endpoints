package authenticator

import (
	"fmt"
	"net/http"
	"os"
	"time"

	cfclient "github.com/cloudfoundry-community/go-cfclient"
)

type Authenticator interface {
	Authenticate(username, password string) (User, error)
}

type BasicAuthenticator struct {
	cfURL      string
	httpClient *http.Client
}

func NewBasicAuthenticator(cfURL string, httpClient *http.Client) *BasicAuthenticator {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 10 * time.Second,
		}
	}
	return &BasicAuthenticator{cfURL, httpClient}
}

func (a *BasicAuthenticator) Authenticate(username, password string) (User, error) {
	cfClient, err := cfclient.NewClient(&cfclient.Config{
		ApiAddress:        a.cfURL,
		Username:          username,
		Password:          password,
		SkipSslValidation: os.Getenv("CF_SKIP_SSL_VALIDATION") == "true",
		UserAgent:         os.Getenv("CF_USER_AGENT"),
		HttpClient:        a.httpClient,
	})
	if err != nil {
		return nil, fmt.Errorf("error authenticating user: %v", err)
	}
	return &BasicUser{cfClient, username}, nil
}

var _ Authenticator = (*BasicAuthenticator)(nil)
