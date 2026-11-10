package crawler

import (
	"context"
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

	if page.Status == statusError {
		return page, liveLinks, true
	}

	liveLinks = crw.parsePage(ctx, &page, resp)

	return page, liveLinks, true
}
