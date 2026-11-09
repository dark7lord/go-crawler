package crawler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// minRetryInterval is the minimum pause between retry attempts.
const minRetryInterval = 100 * time.Millisecond

type fetcher struct {
	client          *http.Client
	requestInterval time.Duration
	retryInterval   time.Duration
	retries         uint
	userAgent       string
	timeout         time.Duration
	limiter         *requestLimiter
}

type requestLimiter struct {
	mu        sync.Mutex
	lastStart time.Time
}

type fetcherConfig struct {
	// client is required. A zero delay applies DefaultDelay; delay and timeout
	// must be non-negative.
	client    *http.Client
	delay     time.Duration
	retries   uint
	userAgent string
	timeout   time.Duration
}

func newFetcher(cfg fetcherConfig) fetcher {
	requestInterval := cfg.delay
	if requestInterval == 0 {
		requestInterval = DefaultDelay
	}

	return fetcher{
		client:          cfg.client,
		requestInterval: requestInterval,
		retryInterval:   max(requestInterval, minRetryInterval),
		retries:         cfg.retries,
		userAgent:       cfg.userAgent,
		timeout:         cfg.timeout,
		limiter:         &requestLimiter{},
	}
}

type cancelBody struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (body *cancelBody) Read(p []byte) (int, error) {
	n, err := body.ReadCloser.Read(p)
	if err != nil {
		body.cancel()
	}

	return n, err
}

func (body *cancelBody) Close() error {
	body.cancel()
	return body.ReadCloser.Close()
}

func isRetryable(res *http.Response, err error) bool {
	if err != nil {
		return !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded)
	}
	code := res.StatusCode

	return (code >= 500 && code <= 599) ||
		code == http.StatusRequestTimeout ||
		code == http.StatusTooManyRequests
}

func (ftr *fetcher) wait(ctx context.Context, interval time.Duration) error {
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

func (ftr *fetcher) get(ctx context.Context, u *url.URL) (*http.Response, error) {
	return ftr.request(ctx, http.MethodGet, u)
}

func (ftr *fetcher) headWithFallback(ctx context.Context, u *url.URL) (*http.Response, error) {
	resp, err := ftr.request(ctx, http.MethodHead, u)
	if err == nil && resp.StatusCode != http.StatusMethodNotAllowed &&
		resp.StatusCode != http.StatusNotImplemented {
		return resp, nil
	}
	if ctx.Err() != nil {
		if resp != nil {
			_ = resp.Body.Close()
		}

		return nil, ctx.Err()
	}
	if resp != nil {
		_ = resp.Body.Close()
	}

	return ftr.get(ctx, u)
}

func (ftr *fetcher) request(ctx context.Context, method string, u *url.URL) (*http.Response, error) {
	remainingRetries := ftr.retries
	interval := ftr.requestInterval
	for {
		if err := ftr.wait(ctx, interval); err != nil {
			return nil, err
		}

		res, err := ftr.fetchOnce(ctx, method, u)

		if !isRetryable(res, err) || remainingRetries == 0 {
			return res, err
		}

		if res != nil {
			_ = res.Body.Close()
		}
		remainingRetries--
		interval = ftr.retryInterval
	}
}

func (ftr *fetcher) fetchOnce(ctx context.Context, method string, u *url.URL) (*http.Response, error) {
	attemptCtx, cancel := ctx, func() {}
	if ftr.timeout > 0 {
		attemptCtx, cancel = context.WithTimeout(ctx, ftr.timeout)
	}

	req, err := http.NewRequestWithContext(attemptCtx, method, u.String(), nil)
	if err != nil {
		cancel()
		return nil, err
	}
	if ftr.userAgent != "" {
		req.Header.Set("User-Agent", ftr.userAgent)
	}

	res, err := ftr.client.Do(req)
	if err != nil {
		cancel()
		return res, err
	}

	res.Body = &cancelBody{ReadCloser: res.Body, cancel: cancel}

	return res, nil
}
