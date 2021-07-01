package spaces_fetcher

import (
	"context"
	"fmt"
	"sync"
	"time"

	"code.cloudfoundry.org/lager"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
)

type SpacesStore interface {
	GetSpaces() []cfclient.Space
}

type SpacesFetcher struct {
	Spaces   []cfclient.Space
	schedule time.Duration
	logger   lager.Logger
	cfClient cfclient.CloudFoundryClient
	mu       sync.Mutex
}

func NewSpacesFetcher(
	schedule time.Duration,
	logger lager.Logger,
	cfClient cfclient.CloudFoundryClient,
) *SpacesFetcher {
	logger = logger.Session("spaces-fetcher")
	return &SpacesFetcher{
		schedule:    schedule,
		logger:      logger,
		cfClient:    cfClient,
	}
}

func (fetcher *SpacesFetcher) GetSpaces() []cfclient.Space {
	fetcher.mu.Lock()
	defer fetcher.mu.Unlock()
	return fetcher.Spaces
}

func (fetcher *SpacesFetcher) Run(ctx context.Context) error {
	loggerSession := fetcher.logger.Session("run")

	loggerSession.Info("start")
	defer loggerSession.Info("end")

	err := fetcher.updateSpaces()
	if err != nil {
		return fmt.Errorf("error initialising list of spaces: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			loggerSession.Info("done")
			return nil
		case <-time.After(fetcher.schedule):
			err = fetcher.updateSpaces()
			if err != nil {
				return fmt.Errorf("error updating list of spaces: %v", err)
			}
		}
	}
}

func (fetcher *SpacesFetcher) updateSpaces() error {
	// FIXME: Relax this locking so it is short-lived
	fetcher.mu.Lock()
	defer fetcher.mu.Unlock()

	spaces, err := fetcher.cfClient.ListSpaces()
	if err != nil {
		return fmt.Errorf("error fetching spaces: %v", err)
	}
	if spaces == nil {
		return fmt.Errorf("list of spaces was nil")
	}

	fetcher.logger.Info("updated-spaces", lager.Data{
		"number-of-spaces": len(spaces),
	})
	fetcher.logger.Debug("updated-spaces-list", lager.Data{
		"spaces": spaces,
	})

	fetcher.Spaces = spaces
	return nil
}

var _ SpacesStore = (*SpacesFetcher)(nil)
