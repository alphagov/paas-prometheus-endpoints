package orgs_fetcher

import (
	"context"
	"fmt"
	"sync"
	"time"

	"code.cloudfoundry.org/lager"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
)

type OrgsStore interface {
	GetOrgs() []cfclient.Org
}

type OrgsFetcher struct {
	Orgs     []cfclient.Org
	schedule time.Duration
	logger   lager.Logger
	cfClient cfclient.CloudFoundryClient
	mu       sync.Mutex
}

func NewOrgsFetcher(
	schedule time.Duration,
	logger lager.Logger,
	cfClient cfclient.CloudFoundryClient,
) *OrgsFetcher {
	logger = logger.Session("orgs-fetcher")
	return &OrgsFetcher{
		schedule:    schedule,
		logger:      logger,
		cfClient:    cfClient,
	}
}

func (fetcher *OrgsFetcher) GetOrgs() []cfclient.Org {
	fetcher.mu.Lock()
	defer fetcher.mu.Unlock()
	return fetcher.Orgs
}

func (fetcher *OrgsFetcher) Run(ctx context.Context) error {
	loggerSession := fetcher.logger.Session("run")

	loggerSession.Info("start")
	defer loggerSession.Info("end")

	err := fetcher.updateOrgs()
	if err != nil {
		return fmt.Errorf("error initialising list of orgs: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			loggerSession.Info("done")
			return nil
		case <-time.After(fetcher.schedule):
			err = fetcher.updateOrgs()
			if err != nil {
				return fmt.Errorf("error updating list of orgs: %v", err)
			}
		}
	}
}

func (fetcher *OrgsFetcher) updateOrgs() error {
	// FIXME: Relax this locking so it is short-lived
	fetcher.mu.Lock()
	defer fetcher.mu.Unlock()

	orgs, err := fetcher.cfClient.ListOrgs()
	if err != nil {
		return fmt.Errorf("error fetching orgs: %v", err)
	}
	if orgs == nil {
		return fmt.Errorf("list of orgs was nil")
	}

	fetcher.logger.Info("updated-orgs", lager.Data{
		"number-of-orgs": len(orgs),
	})
	fetcher.logger.Debug("updated-orgs-list", lager.Data{
		"orgs": orgs,
	})

	fetcher.Orgs = orgs
	return nil
}

var _ OrgsStore = (*OrgsFetcher)(nil)
