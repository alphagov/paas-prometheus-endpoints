package authenticator

import (
	"fmt"
	"net/url"
	"strings"

	cfclient "github.com/cloudfoundry-community/go-cfclient"
)

type CFUser interface {
	Username() string
	ListServiceInstancesMatchingPlanGUIDs(planGuids []string) ([]cfclient.ServiceInstance, error)
}

type BasicCFUser struct {
	cfClient cfclient.CloudFoundryClient
	username string
}

func (u BasicCFUser) Username() string {
	return u.username
}

func (u BasicCFUser) ListServiceInstancesMatchingPlanGUIDs(servicePlanGuids []string) ([]cfclient.ServiceInstance, error) {
	q := url.Values{}
	q.Add("q", fmt.Sprintf("service_plan_guid IN %s", strings.Join(servicePlanGuids, ",")))
	serviceInstances, err := u.cfClient.ListServiceInstancesByQuery(q)
	if err != nil {
		return nil, fmt.Errorf("error listing service instances: %v", err)
	}
	return serviceInstances, nil
}
