package fetch

import (
	"context"
	"net/http"
	"net/url"
)

// Head checks a resource with HEAD and falls back to GET when HEAD is
// unsupported.
func (ftr *Fetcher) Head(ctx context.Context, u *url.URL) (*http.Response, error) {
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

	return ftr.Get(ctx, u)
}
