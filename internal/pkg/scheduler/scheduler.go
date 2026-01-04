package scheduler

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"
)

type Runnable interface {
	Name() string
	Run(ctx context.Context) error
}

type Scheduler struct {
	Interval    time.Duration
	Jitter      time.Duration
	Concurrency int  // max concurrent jobs per tick
	NoOverlap   bool // skip tick when previous tick is still running
	StopTimeout time.Duration
}

func (s *Scheduler) Start(ctx context.Context, jobs []Runnable) <-chan error {
	if ctx == nil {
		ctx = context.Background()
	}

	out := make(chan error, 64)
	errs := make(chan error, 256)

	interval := s.Interval
	if interval <= 0 {
		interval = 10 * time.Minute
	}

	concurrency := s.Concurrency
	if concurrency <= 0 {
		concurrency = len(jobs)
	}
	if concurrency <= 0 {
		concurrency = 1
	}

	jitter := s.Jitter
	noOverlap := s.NoOverlap

	stopTimeout := s.StopTimeout
	if stopTimeout <= 0 {
		stopTimeout = 5 * time.Second
	}

	var running atomic.Bool
	var ticksWG sync.WaitGroup

	runTick := func() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if noOverlap && !running.CompareAndSwap(false, true) {
			return
		}

		ticksWG.Go(func() {
			defer func() {
				if noOverlap {
					running.Store(false)
				}
				if r := recover(); r != nil {
					select {
					case errs <- fmt.Errorf("tick panic: %v", r):
					default:
					}
				}
			}()

			if jitter > 0 {
				d := time.Duration(rand.Int64N(int64(jitter)))
				timer := time.NewTimer(d)
				defer timer.Stop()

				select {
				case <-ctx.Done():
					return
				case <-timer.C:
				}
			}

			g, gctx := errgroup.WithContext(ctx)
			g.SetLimit(concurrency)

			for _, job := range jobs {
				g.Go(func() error {
					defer func() {
						if r := recover(); r != nil {
							select {
							case errs <- fmt.Errorf("%s panic: %v", job.Name(), r):
							default:
							}
						}
					}()

					if err := job.Run(gctx); err != nil {
						select {
						case errs <- fmt.Errorf("%s: %w", job.Name(), err):
						default:
						}
					}
					return nil
				})
			}

			_ = g.Wait()
		})
	}

	go func() {
		runTick()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runTick()
			}
		}
	}()

	go func() {
		defer close(out)

		var ticksDone <-chan struct{}
		var stopC <-chan time.Time

		var stopTimer *time.Timer
		defer func() {
			if stopTimer != nil {
				stopTimer.Stop()
			}
		}()

		for {
			select {
			case err := <-errs:
				if err == nil {
					continue
				}
				select {
				case out <- err:
				default:
				}

			case <-ctx.Done():
				if ticksDone == nil {
					done := make(chan struct{})
					ticksDone = done
					go func() {
						ticksWG.Wait()
						close(done)
					}()

					stopTimer = time.NewTimer(stopTimeout)
					stopC = stopTimer.C
				}

			case <-ticksDone:
				return

			case <-stopC:
				return
			}
		}
	}()

	return out
}
