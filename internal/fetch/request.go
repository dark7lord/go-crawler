package fetch

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
)

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

// Get performs a GET request, retrying retryable failures.
func (ftr *Fetcher) Get(ctx context.Context, u *url.URL) (*http.Response, error) {
	return ftr.request(ctx, http.MethodGet, u)
}

func (ftr *Fetcher) request(ctx context.Context, method string, u *url.URL) (*http.Response, error) {
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

func (ftr *Fetcher) fetchOnce(ctx context.Context, method string, u *url.URL) (*http.Response, error) {
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
