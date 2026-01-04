package feeds

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"

	"reader/internal/app/reader/feeds/common"
	"reader/internal/pkg/downloader"
	"reader/internal/pkg/scheduler"
)

const (
	concurrency  = 4
	httpTimeout  = 20 * time.Second
	maxPoolSize  = 8
	pollInterval = 10 * time.Minute
	pollJitter   = 10 * time.Second
	stopTimeout  = 5 * time.Second
)

// PollFeeds start poll of all feeds
func PollFeeds(parent context.Context, log *logrus.Logger) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}

	parent, cancel := context.WithCancel(parent)

	deps := common.Deps{
		HTTP:     downloader.New(httpTimeout),
		HTTPPool: semaphore.NewWeighted(maxPoolSize),
		Log:      log,
	}

	s := scheduler.Scheduler{
		Interval:    pollInterval,
		Jitter:      pollJitter,
		Concurrency: concurrency,
		NoOverlap:   true,
		StopTimeout: stopTimeout,
	}

	errCh := s.Start(parent, loadJobs(deps))
	done := make(chan struct{})
	go func() {
		defer close(done)
		for err := range errCh {
			log.Error(err)
		}
	}()

	stop := func() {
		cancel()
		<-done
	}

	return parent, stop
}
