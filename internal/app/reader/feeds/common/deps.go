package common

import (
	"context"
	"net/http"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"

	"reader/internal/app/reader/models"
	"reader/internal/pkg/downloader"
)

type Deps struct {
	HTTP downloader.Doer
	Repo *models.Repo

	HTTPPool *semaphore.Weighted
	Log      *logrus.Logger
}

func (d Deps) Do(ctx context.Context, req *http.Request) (*http.Response, func(), error) {
	release := func() {}

	if d.HTTPPool != nil {
		if err := d.HTTPPool.Acquire(ctx, 1); err != nil {
			return nil, nil, err
		}
		release = func() {
			d.HTTPPool.Release(1)
		}
	}

	resp, err := d.HTTP.Do(req)
	if err != nil {
		release()
		return nil, nil, err
	}

	return resp, release, nil
}
