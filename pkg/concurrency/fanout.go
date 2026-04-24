package concurrency

import (
	"context"
	"runtime"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

type Task func(ctx context.Context) error

type FanoutOptions struct {
	MaxWorkers int
}

var DefaultFanout = FanoutOptions{MaxWorkers: 4}

func Run(ctx context.Context, tasks []Task, opts FanoutOptions) error {
	if len(tasks) == 0 {
		return nil
	}
	maxWorkers := opts.MaxWorkers
	if maxWorkers <= 0 {
		maxWorkers = runtime.GOMAXPROCS(0)
		if maxWorkers <= 0 {
			maxWorkers = 1
		}
	}

	g, gctx := errgroup.WithContext(ctx)
	sem := semaphore.NewWeighted(int64(maxWorkers))

	for _, task := range tasks {
		task := task
		g.Go(func() error {
			if err := sem.Acquire(gctx, 1); err != nil {
				return err
			}
			defer sem.Release(1)
			return task(gctx)
		})
	}

	return g.Wait()
}
