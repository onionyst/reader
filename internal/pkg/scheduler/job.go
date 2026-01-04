package scheduler

import (
	"context"
)

type Job struct {
	name string
	fn   func(ctx context.Context) error
}

func (j *Job) Name() string {
	return j.name
}

func (j *Job) Run(ctx context.Context) error {
	return j.fn(ctx)
}

func NewJob(name string, fn func(ctx context.Context) error) Runnable {
	return &Job{
		name: name,
		fn:   fn,
	}
}
