package service_plans_fetcher

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"time"

	"code.cloudfoundry.org/lager"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
)

type ServicePlansStore interface {
	GetService() *cfclient.Service
	GetServicePlans() []cfclient.ServicePlan
}

type ServicePlansFetcher struct {
	serviceName  string
	service      *cfclient.Service
	servicePlans []cfclient.ServicePlan
	schedule     time.Duration
	logger       lager.Logger
	cfClient     cfclient.CloudFoundryClient
	mu           sync.Mutex
}

func NewServicePlansFetcher(
	serviceName string,
	schedule time.Duration,
	logger lager.Logger,
	cfClient cfclient.CloudFoundryClient,
) *ServicePlansFetcher {
	logger = logger.Session("service-plans-fetcher")
	return &ServicePlansFetcher{
		serviceName: serviceName,
		schedule:    schedule,
		logger:      logger,
		cfClient:    cfClient,
	}
}

func (f *ServicePlansFetcher) GetService() *cfclient.Service {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.service
}

func (f *ServicePlansFetcher) GetServicePlans() []cfclient.ServicePlan {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.servicePlans
}

func (f *ServicePlansFetcher) Run(ctx context.Context) error {
	lsession := f.logger.Session("run")

	lsession.Info("start")
	defer lsession.Info("end")

	err := f.updateServicePlans()
	if err != nil {
		return fmt.Errorf("error initialising list of service plans: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			lsession.Info("done")
			return nil
		case <-time.After(f.schedule):
			err = f.updateServicePlans()
			if err != nil {
				return fmt.Errorf("error updating list of service plans: %v", err)
			}
		}
	}
}

func (f *ServicePlansFetcher) updateServicePlans() error {
	// FIXME: Relax this locking so it is short-lived
	f.mu.Lock()
	defer f.mu.Unlock()

	q1 := url.Values{}
	q1.Add("q", fmt.Sprintf("label:%s", f.serviceName))
	services, err := f.cfClient.ListServicesByQuery(q1)
	if err != nil {
		return fmt.Errorf("error fetching service named '%s': %v", f.serviceName, err)
	}
	if len(services) != 1 {
		return fmt.Errorf("unexpected number of results when fetching service named '%s': %d results", f.serviceName, len(services))
	}
	f.service = &services[0]

	q2 := url.Values{}
	q2.Add("q", fmt.Sprintf("service_guid:%s", f.service.Guid))
	servicePlans, err := f.cfClient.ListServicePlansByQuery(q2)
	if err != nil {
		return fmt.Errorf("error fetching service plans: %v", err)
	}
	if servicePlans == nil {
		return fmt.Errorf("list of service plans was nil")
	}

	f.logger.Info("updated-service-plans", lager.Data{
		"number-of-service-plans": len(servicePlans),
	})
	f.logger.Debug("updated-service-plans-list", lager.Data{
		"service-plans": servicePlans,
	})

	f.servicePlans = servicePlans
	return nil
}

var _ ServicePlansStore = (*ServicePlansFetcher)(nil)
