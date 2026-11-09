package fetch

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFetcherRateLimit(t *testing.T) {
	const shortDelay = 20 * time.Millisecond

	tests := []struct {
		name     string
		delay    time.Duration
		requests int
		wantMin  time.Duration
		wantMax  time.Duration
	}{
		{
			name:     "first request is not delayed",
			delay:    50 * time.Millisecond,
			requests: 1,
			wantMax:  50 * time.Millisecond,
		},
		{
			name:     "spaces requests by explicit delay",
			delay:    shortDelay,
			requests: 5,
			wantMin:  4 * shortDelay,
			wantMax:  200 * time.Millisecond,
		},
		{
			name:     "spaces requests by default when delay is zero",
			requests: 5,
			wantMin:  4 * DefaultDelay,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ftr, u := newTestFetcher(t, tt.delay)
			start := time.Now()

			for range tt.requests {
				resp, err := ftr.Get(t.Context(), u)
				require.NoError(t, err)
				require.NoError(t, resp.Body.Close())
			}

			elapsed := time.Since(start)
			if tt.wantMin > 0 {
				require.GreaterOrEqual(t, elapsed, tt.wantMin)
			}
			if tt.wantMax > 0 {
				require.Less(t, elapsed, tt.wantMax)
			}
		})
	}
}

func TestFetcherCancelStopsWaiting(t *testing.T) {
	ftr, u := newTestFetcher(t, 5*time.Second)
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()

	resp, err := ftr.Get(ctx, u)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	start := time.Now()
	resp, err = ftr.Get(ctx, u)
	if resp != nil {
		_ = resp.Body.Close()
	}

	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Nil(t, resp)
	require.Less(t, time.Since(start), time.Second)
}
