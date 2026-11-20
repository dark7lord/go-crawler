package crawler_test

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"code/crawler"
)

func findPage(t *testing.T, report crawler.Report, pageURL string) crawler.Page {
	t.Helper()

	for _, page := range report.Pages {
		if page.URL == pageURL {
			return page
		}
	}

	t.Fatalf("page %q not found in report", pageURL)

	return crawler.Page{}
}

func TestPageRedirectOutOfScope(t *testing.T) {
	var internalURL string

	externalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/landing":
			_, _ = fmt.Fprintf(
				w,
				`<title>External title</title>
				<h1>External page</h1>
				<a href="%s/secret">internal link</a>
				<img src="/evil.png">`,
				internalURL,
			)
		case "/evil.png":
			_, _ = fmt.Fprint(w, "evil")
		default:
			http.NotFound(w, r)
		}
	})

	external := newTestServer(t, externalHandler)

	internalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			_, _ = fmt.Fprint(w, `<a href="/move">move</a>`)
		case "/move":
			http.Redirect(w, r, external.URL+"/landing", http.StatusFound)
		case "/secret":
			_, _ = fmt.Fprint(w, `<title>Secret</title>`)
		default:
			http.NotFound(w, r)
		}
	})

	internal := newTestServer(t, internalHandler)
	internalURL = internal.URL

	report := analyzeReport(t, crawler.Options{
		URL:        internal.URL,
		Depth:      3,
		HTTPClient: internal.Client(),
	})

	move := findPage(t, report, internal.URL+"/move")
	require.Equal(t, "error", move.Status)
	require.Equal(t, http.StatusOK, move.HTTPStatus)
	require.Contains(t, move.Error, "redirected out of scope")
	require.Contains(t, move.Error, external.URL+"/landing")
	require.Zero(t, move.SEO)
	require.Nil(t, move.BrokenLinks)
	require.Nil(t, move.Assets)

	urls := reportURLs(report)
	require.NotContains(t, urls, external.URL+"/landing")
	require.NotContains(t, urls, internal.URL+"/secret")
}

func TestPageRedirectOutOfScopeWithErrorStatus(t *testing.T) {
	external := newTestServer(t, http.NotFoundHandler())

	internalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			http.Redirect(w, r, external.URL+"/missing", http.StatusMovedPermanently)
		default:
			http.NotFound(w, r)
		}
	})

	internal := newTestServer(t, internalHandler)

	page := firstPage(t, crawler.Options{
		URL:        internal.URL,
		Depth:      1,
		HTTPClient: internal.Client(),
	})

	require.Equal(t, "error", page.Status)
	require.Equal(t, http.StatusNotFound, page.HTTPStatus)
	require.Contains(t, page.Error, "redirected out of scope")
	require.Contains(t, page.Error, external.URL+"/missing")
	require.Zero(t, page.SEO)
	require.Nil(t, page.Assets)
}

func TestPageRedirectInScopeIsParsed(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/old":
			http.Redirect(w, r, "/new", http.StatusFound)
		case "/new":
			_, _ = fmt.Fprint(w, `<title>New page</title>`)
		default:
			http.NotFound(w, r)
		}
	})

	server := newTestServer(t, handler)

	page := firstPage(t, crawler.Options{
		URL:        server.URL + "/old",
		Depth:      1,
		HTTPClient: server.Client(),
	})

	require.Equal(t, server.URL+"/old", page.URL)
	require.Equal(t, "ok", page.Status)
	require.Equal(t, http.StatusOK, page.HTTPStatus)
	require.Empty(t, page.Error)
	require.True(t, page.SEO.HasTitle)
	require.Equal(t, "New page", page.SEO.Title)
}

func TestPageRedirectWithoutLocation(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusFound)
		_, _ = fmt.Fprint(w, `<title>Not a page</title>`)
	})

	server := newTestServer(t, handler)

	page := firstPage(t, crawler.Options{
		URL:        server.URL,
		Depth:      1,
		HTTPClient: server.Client(),
	})

	require.Equal(t, "error", page.Status)
	require.Equal(t, http.StatusFound, page.HTTPStatus)
	require.Contains(t, page.Error, "redirect not followed")
	require.Contains(t, page.Error, "HTTP 302 Found")
	require.Zero(t, page.SEO)
	require.Nil(t, page.BrokenLinks)
	require.Nil(t, page.Assets)
}

func TestPageRedirectNotFollowedByCustomClient(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			http.Redirect(w, r, "/new", http.StatusFound)
		case "/new":
			_, _ = fmt.Fprint(w, `<title>New page</title>`)
		default:
			http.NotFound(w, r)
		}
	})

	server := newTestServer(t, handler)

	client := server.Client()
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}

	page := firstPage(t, crawler.Options{
		URL:        server.URL,
		Depth:      1,
		HTTPClient: client,
	})

	require.Equal(t, "error", page.Status)
	require.Equal(t, http.StatusFound, page.HTTPStatus)
	require.Contains(t, page.Error, "redirect not followed")
	require.Contains(t, page.Error, "HTTP 302 Found")
	require.Zero(t, page.SEO)
	require.Nil(t, page.Assets)
}

func TestPageUnfollowedRedirectIsNotRetried(t *testing.T) {
	var requests atomic.Int32

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusFound)
	})

	server := newTestServer(t, handler)

	page := firstPage(t, crawler.Options{
		URL:        server.URL,
		Depth:      1,
		Retries:    2,
		HTTPClient: server.Client(),
	})

	require.Equal(t, "error", page.Status)
	require.Equal(t, int32(1), requests.Load())
}
