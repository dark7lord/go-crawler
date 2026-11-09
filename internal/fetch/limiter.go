package fetch

import (
	"context"
	"sync"
	"time"
)

type requestLimiter struct {
	mu        sync.Mutex
	lastStart time.Time
}

func (ftr *Fetcher) wait(ctx context.Context, interval time.Duration) error {
	limiter := ftr.limiter
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}

	if !limiter.lastStart.IsZero() {
		earliest := limiter.lastStart.Add(interval)
		if now := time.Now(); now.Before(earliest) {
			timer := time.NewTimer(time.Until(earliest))
			defer timer.Stop()

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
			}
		}
	}

	limiter.lastStart = time.Now()

	return nil
}
