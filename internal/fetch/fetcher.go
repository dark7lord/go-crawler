// Package fetch performs HTTP requests with rate limiting and retries.
package fetch

import (
	"net/http"
	"time"
)

// DefaultDelay is the request interval used when Config.Delay is zero.
const DefaultDelay = 100 * time.Millisecond

// minRetryInterval is the minimum pause between retry attempts.
const minRetryInterval = 100 * time.Millisecond

// Fetcher performs HTTP requests with shared rate limiting and retries.
type Fetcher struct {
	client          *http.Client
	requestInterval time.Duration
	retryInterval   time.Duration
	retries         uint
	userAgent       string
	timeout         time.Duration
	limiter         *requestLimiter
}

// Config configures a Fetcher.
type Config struct {
	// Client is required.
	Client *http.Client
	// Delay is the interval between request starts. Zero applies DefaultDelay.
	// Negative values must be rejected by the caller.
	Delay time.Duration
	// Retries is the number of additional attempts for a retryable failure.
	Retries uint
	// UserAgent overrides the User-Agent header when non-empty.
	UserAgent string
	// Timeout limits one HTTP attempt. Zero preserves the client's timeout.
	Timeout time.Duration
}

// New creates a Fetcher. A zero Delay applies DefaultDelay.
func New(cfg Config) *Fetcher {
	requestInterval := cfg.Delay
	if requestInterval == 0 {
		requestInterval = DefaultDelay
	}

	return &Fetcher{
		client:          cfg.Client,
		requestInterval: requestInterval,
		retryInterval:   max(requestInterval, minRetryInterval),
		retries:         cfg.Retries,
		userAgent:       cfg.UserAgent,
		timeout:         cfg.Timeout,
		limiter:         &requestLimiter{},
	}
}
