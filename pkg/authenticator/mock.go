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
		return &MockUser{MockUsername: username}, nil
	}
	return nil, fmt.Errorf("password not allowed")
}

var _ Authenticator = (*MockAuthenticator)(nil)

type MockUser struct {
	MockUsername            string
	MockServiceInstances    []cfclient.ServiceInstance
	MockServiceInstancesErr error
}

func (u *MockUser) Username() string {
	return u.MockUsername
}

func (u *MockUser) ListServiceInstancesMatchingPlanGUIDs(planGuids []string) ([]cfclient.ServiceInstance, error) {
	if u.MockServiceInstancesErr != nil {
		return nil, u.MockServiceInstancesErr
	}
	matchingServiceInstances := []cfclient.ServiceInstance{}
	for _, serviceInstance := range u.MockServiceInstances {
		matches := false
		for _, planGuid := range planGuids {
			if serviceInstance.ServicePlanGuid == planGuid {
				matches = true
				break
			}
		}
		if matches {
			matchingServiceInstances = append(matchingServiceInstances, serviceInstance)
		}
	}

	return matchingServiceInstances, nil
}

var _ User = (*MockUser)(nil)
