// Package urlutil normalizes and resolves crawl URLs.
package urlutil

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// ErrInvalid is returned when the URL can't be crawled.
var ErrInvalid = errors.New("invalid url")

// Normalize adds http:// if needed and validates an http(s) URL.
func Normalize(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}

	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalid, err)
	}

	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("%w: %q", ErrInvalid, raw)
	}

	return u, nil
}

// Resolve makes href absolute against base. The false result means the link
// should be skipped.
func Resolve(base *url.URL, href string) (*url.URL, bool) {
	trimmed := strings.TrimSpace(href)
	if trimmed == "" {
		return nil, false
	}

	if strings.HasPrefix(trimmed, "#") {
		return nil, false
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return nil, false
	}

	u := base.ResolveReference(parsed)

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, false
	}

	return u, true
}
