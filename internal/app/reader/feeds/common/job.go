package common

import (
	"context"

	"reader/internal/pkg/scheduler"
)

type RSSJob interface {
	Name() string
	Run(ctx context.Context, d Deps) error
}

func Bind(d Deps, job RSSJob) scheduler.Runnable {
	return scheduler.NewJob(job.Name(), func(ctx context.Context) error {
		return job.Run(ctx, d)
	})
}
