package fetch

import (
	"context"
	"net/http"
	"net/url"
)

// Head checks a resource with HEAD and confirms any failed HEAD with GET,
// because servers may serve HEAD and GET differently.
func (ftr *Fetcher) Head(ctx context.Context, u *url.URL) (*http.Response, error) {
	resp, err := ftr.request(ctx, http.MethodHead, u)
	if err == nil && resp.StatusCode < http.StatusBadRequest {
		return resp, nil
	}

	if resp != nil {
		_ = resp.Body.Close()
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	return ftr.Get(ctx, u)
}
