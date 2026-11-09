// Package crawler implements website structure analysis.
package crawler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"code/internal/fetch"
	"code/internal/urlutil"
)

// DefaultDelay is the request interval used when Options.Delay is zero.
const DefaultDelay = fetch.DefaultDelay

// ErrInvalidDelay is returned when Options.Delay is negative.
var ErrInvalidDelay = errors.New("invalid delay")

// ErrInvalidTimeout is returned when Options.Timeout is negative.
var ErrInvalidTimeout = errors.New("invalid timeout")

// ErrInvalidURL is returned when the URL can't be crawled.
var ErrInvalidURL = urlutil.ErrInvalid

// Options configures a crawl run.
type Options struct {
	URL string
	// Depth is the maximum number of page levels to visit. The start page is level 1.
	Depth   uint
	Retries uint
	// Delay is the minimal pause between the starts of two requests,
	// shared by the whole crawl. A positive value is always used as-is,
	// including values below DefaultDelay. Zero falls back to DefaultDelay;
	// negative values are invalid.
	Delay time.Duration
	// Timeout is the maximum duration of one HTTP attempt, including reading
	// its response body. Zero preserves the HTTP client's configured timeout;
	// negative values are invalid.
	Timeout   time.Duration
	UserAgent string
	// Concurrency is the maximum number of pages processed in parallel.
	// Zero means one worker.
	Concurrency uint
	IndentJSON  bool
	HTTPClient  *http.Client
}

func (opts Options) validate() error {
	if opts.Delay < 0 {
		return fmt.Errorf("%w: %s", ErrInvalidDelay, opts.Delay)
	}
	if opts.Timeout < 0 {
		return fmt.Errorf("%w: %s", ErrInvalidTimeout, opts.Timeout)
	}

	return nil
}

// Analyze crawls a website and returns a report about its structure.
func Analyze(ctx context.Context, opts Options) (data []byte, err error) {
	crw, err := newCrawler(opts)
	if err != nil {
		return nil, err
	}

	crw.crawl(ctx)

	return crw.report()
}
