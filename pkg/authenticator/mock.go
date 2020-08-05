package authenticator

import (
	"fmt"

	cfclient "github.com/cloudfoundry-community/go-cfclient"
)

type MockAuthenticator struct {
	AllowedUsername string
	AllowedPassword string
}

func (a *MockAuthenticator) Authenticate(username, password string) (User, error) {
	if username == a.AllowedUsername && password == a.AllowedPassword {
		return &MockUser{username}, nil
	}
	return nil, fmt.Errorf("password not allowed")
}

var _ Authenticator = (*MockAuthenticator)(nil)

type MockUser struct {
	username string
}

func (u *MockUser) Username() string {
	return u.username
}

func (u *MockUser) ListServiceInstancesMatchingPlanGUIDs(planGuids []string) ([]cfclient.ServiceInstance, error) {
	return nil, fmt.Errorf("not implemented on mockuser")
}

var _ User = (*MockUser)(nil)
