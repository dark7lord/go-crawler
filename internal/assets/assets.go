// Package assets discovers and loads resources referenced by pages.
package assets

import (
	"net/url"
	"slices"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"code/internal/urlutil"
)

// Type identifies the kind of a resource referenced by a page.
type Type string

// Asset types.
const (
	TypeImage  Type = "image"
	TypeScript Type = "script"
	TypeStyle  Type = "style"
)

// Ref is a resource referenced by a page.
type Ref struct {
	URL  *url.URL
	Type Type
}

// Asset contains the measured result for a resource referenced by a page.
type Asset struct {
	URL        string `json:"url"`
	Type       Type   `json:"type"`
	StatusCode int    `json:"status_code"`
	SizeBytes  int64  `json:"size_bytes"`
	Error      string `json:"error,omitempty"`
}

// Extract returns unique assets referenced by a document.
func Extract(doc *goquery.Document, pageURL *url.URL) []Ref {
	var assets []Ref
	seen := make(map[string]struct{})

	addAsset := func(_ int, selection *goquery.Selection) {
		reference, ok := extractRef(selection, pageURL)
		if !ok {
			return
		}

		key := reference.URL.String()
		if _, exists := seen[key]; exists {
			return
		}

		seen[key] = struct{}{}
		assets = append(assets, reference)
	}

	selection := doc.Find("img[src], script[src], link[href]")
	selection.Each(addAsset)

	slices.SortStableFunc(assets, compareType)

	return assets
}

func extractRef(sel *goquery.Selection, pageURL *url.URL) (Ref, bool) {
	var (
		rawURL string
		typ    Type
	)

	tag := sel.Get(0).Data
	switch tag {
	case "img":
		rawURL, typ = sel.AttrOr("src", ""), TypeImage
	case "script":
		rawURL, typ = sel.AttrOr("src", ""), TypeScript
	case "link":
		rel := sel.AttrOr("rel", "")
		if !hasRelToken(rel, "stylesheet") {
			return Ref{}, false
		}
		rawURL, typ = sel.AttrOr("href", ""), TypeStyle
	default:
		return Ref{}, false
	}

	absoluteURL, ok := urlutil.Resolve(pageURL, rawURL)
	if !ok {
		return Ref{}, false
	}

	return Ref{URL: absoluteURL, Type: typ}, true
}

func compareType(a, b Ref) int {
	return strings.Compare(string(a.Type), string(b.Type))
}

func hasRelToken(rel, token string) bool {
	tokens := strings.Fields(rel)

	return slices.ContainsFunc(tokens, func(candidate string) bool {
		return strings.EqualFold(candidate, token)
	})
}
