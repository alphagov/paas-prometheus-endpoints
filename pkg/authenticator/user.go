package authenticator

import (
	"fmt"
	"net/url"
	"strings"

	cfclient "github.com/cloudfoundry-community/go-cfclient"
)

type User interface {
	Username() string
	ListServiceInstancesMatchingPlanGUIDs(planGuids []string) ([]cfclient.ServiceInstance, error)
}

type BasicUser struct {
	cfClient cfclient.CloudFoundryClient
	username string
}

func NewBasicUser(cfClient cfclient.CloudFoundryClient, username string) *BasicUser {
	return &BasicUser{cfClient, username}
}

func (u BasicUser) Username() string {
	return u.username
}

func (u BasicUser) ListServiceInstancesMatchingPlanGUIDs(servicePlanGuids []string) ([]cfclient.ServiceInstance, error) {
	q := url.Values{}
	q.Add("q", fmt.Sprintf("service_plan_guid IN %s", strings.Join(servicePlanGuids, ",")))
	serviceInstances, err := u.cfClient.ListServiceInstancesByQuery(q)
	if err != nil {
		return nil, fmt.Errorf("error listing service instances: %v", err)
	}
	return serviceInstances, nil
}

var _ User = (*BasicUser)(nil)
