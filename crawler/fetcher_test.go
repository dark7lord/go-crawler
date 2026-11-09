package crawler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func newTestFetcher(t *testing.T, delay time.Duration) (*fetcher, *url.URL) {
	t.Helper()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("delay: " + delay.String()))
	})

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	u, err := url.Parse(server.URL)
	require.NoError(t, err)

	ftr := newFetcher(fetcherConfig{
		client: server.Client(),
		delay:  delay,
	})

	return &ftr, u
}

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
				resp, err := ftr.get(t.Context(), u)
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

func TestNewFetcherRequestInterval(t *testing.T) {
	tests := []struct {
		name         string
		delay        time.Duration
		wantInterval time.Duration
	}{
		{
			name:         "explicit delay is used as-is",
			delay:        50 * time.Millisecond,
			wantInterval: 50 * time.Millisecond,
		},
		{
			name:         "explicit delay below default is honored",
			delay:        20 * time.Millisecond,
			wantInterval: 20 * time.Millisecond,
		},
		{
			name:         "zero delay falls back to default",
			wantInterval: DefaultDelay,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ftr, _ := newTestFetcher(t, tt.delay)

			require.Equal(t, tt.wantInterval, ftr.requestInterval)
		})
	}
}

func TestNewFetcherRetryInterval(t *testing.T) {
	tests := []struct {
		name         string
		delay        time.Duration
		wantInterval time.Duration
	}{
		{
			name:         "short explicit delay keeps minimum retry interval",
			delay:        50 * time.Millisecond,
			wantInterval: minRetryInterval,
		},
		{
			name:         "zero delay uses minimum retry interval",
			wantInterval: minRetryInterval,
		},
		{
			name:         "long explicit delay raises retry interval",
			delay:        500 * time.Millisecond,
			wantInterval: 500 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ftr, _ := newTestFetcher(t, tt.delay)

			require.Equal(t, tt.wantInterval, ftr.retryInterval)
		})
	}
}

func TestFetcherCancelStopsWaiting(t *testing.T) {
	ftr, u := newTestFetcher(t, 5*time.Second)
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()

	resp, err := ftr.get(ctx, u)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	start := time.Now()
	resp, err = ftr.get(ctx, u)
	if resp != nil {
		_ = resp.Body.Close()
	}

	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Nil(t, resp)
	require.Less(t, time.Since(start), time.Second)
}
