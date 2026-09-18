// Package parallel contains small bounded-concurrency helpers used by the
// component build and packaging services.
package parallel

import (
	"context"
	"sync"
	"sync/atomic"
)

// DefaultWorkers is the maximum number of components processed at once.
const DefaultWorkers = 5

// ForEach processes indexes with a bounded number of workers. Returning false
// from work stops new queued indexes from starting, while work that has
// already started is allowed to finish. Cancellation stops queued work.
func ForEach(ctx context.Context, indexes []int, limit int, work func(index int) bool) {
	if len(indexes) == 0 {
		return
	}
	if limit < 1 {
		limit = 1
	}
	if limit > len(indexes) {
		limit = len(indexes)
	}

	jobs := make(chan int)
	var stopped atomic.Bool
	var workers sync.WaitGroup
	workers.Add(limit)
	for worker := 0; worker < limit; worker++ {
		go func() {
			defer workers.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case index, ok := <-jobs:
					if !ok {
						return
					}
					if stopped.Load() {
						continue
					}
					if !work(index) {
						stopped.Store(true)
					}
				}
			}
		}()
	}

	for _, index := range indexes {
		if stopped.Load() {
			break
		}
		select {
		case <-ctx.Done():
			break
		case jobs <- index:
		}
		if ctx.Err() != nil {
			break
		}
	}
	close(jobs)
	workers.Wait()
}
