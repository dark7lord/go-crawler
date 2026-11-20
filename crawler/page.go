package crawler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"code/internal/assets"
	"code/internal/htmlparse"
	"code/internal/seo"
)

const (
	statusOK    = "ok"
	statusError = "error"
)

var (
	errOutOfScope          = errors.New("redirected out of scope")
	errRedirectNotFollowed = errors.New("redirect not followed")
)

func newPage(u *url.URL, depth uint) Page {
	return Page{
		URL:          u.String(),
		Depth:        depth,
		DiscoveredAt: time.Now().UTC(),
	}
}

func (p *Page) setError(err error) {
	p.Status = statusError
	p.Error = err.Error()
}

func (p *Page) setHTTPStatus(code int) {
	p.HTTPStatus = code

	status := statusOK
	if code >= http.StatusBadRequest {
		status = statusError
	}

	p.Status = status
}

// unfollowedRedirectError reports a redirect that the HTTP client did not follow.
func unfollowedRedirectError(resp *http.Response) error {
	if !isRedirectStatus(resp.StatusCode) {
		return nil
	}

	return fmt.Errorf(
		"%w: HTTP %d %s",
		errRedirectNotFollowed,
		resp.StatusCode,
		http.StatusText(resp.StatusCode),
	)
}

// isRedirectStatus matches the statuses net/http follows automatically.
func isRedirectStatus(code int) bool {
	switch code {
	case http.StatusMovedPermanently,
		http.StatusFound,
		http.StatusSeeOther,
		http.StatusTemporaryRedirect,
		http.StatusPermanentRedirect:
		return true
	}

	return false
}

func (crw *crawler) parsePage(ctx context.Context, page *Page, resp *http.Response) []foundURL {
	doc, err := htmlparse.Parse(resp.Body)
	if err != nil {
		page.setError(err)
		return nil
	}

	reqURL := resp.Request.URL

	page.SEO = seo.Extract(doc)

	assetRefs := assets.Extract(doc, reqURL)
	page.Assets = crw.assetLoader.Load(ctx, assetRefs)

	links := htmlparse.ExtractLinks(doc)
	brokenLinks, liveLinks := crw.checkLinks(ctx, links, reqURL, page.Depth)
	page.BrokenLinks = brokenLinks
	if page.Depth+1 >= crw.depth {
		return nil
	}

	return liveLinks
}

// validatePageResponse reports whether resp can be parsed as a page.
func (crw *crawler) validatePageResponse(
	page *Page,
	resp *http.Response,
) bool {
	if resp.Request.URL.Host != crw.host {
		page.setError(fmt.Errorf("%w: %s", errOutOfScope, resp.Request.URL))

		return false
	}

	if err := unfollowedRedirectError(resp); err != nil {
		page.setError(err)

		return false
	}

	return page.Status != statusError
}

func (crw *crawler) fetchPage(
	ctx context.Context,
	pageURL *url.URL,
	depth uint,
) (page Page, liveLinks []foundURL, ok bool) {
	page = newPage(pageURL, depth)

	defer func() {
		if r := recover(); r != nil {
			page.setError(fmt.Errorf("panic: %v", r))
			ok = true
		}
	}()

	resp, err := crw.fetcher.Get(ctx, pageURL)
	if err != nil {
		if ctx.Err() != nil {
			return page, liveLinks, false
		}

		page.setError(err)

		return page, liveLinks, true
	}
	defer func() { _ = resp.Body.Close() }()

	page.setHTTPStatus(resp.StatusCode)

	if !crw.validatePageResponse(&page, resp) {
		return page, liveLinks, true
	}

	liveLinks = crw.parsePage(ctx, &page, resp)

	return page, liveLinks, true
}
