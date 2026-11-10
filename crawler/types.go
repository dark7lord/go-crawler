package crawler

import (
	"time"

	"code/internal/assets"
	"code/internal/seo"
)

// Page is a single crawled page in the report.
type Page struct {
	URL          string       `json:"url"`
	Depth        uint         `json:"depth"`
	HTTPStatus   int          `json:"http_status"`
	Status       string       `json:"status"`
	Error        string       `json:"error,omitempty"`
	SEO          SEOResult    `json:"seo"`
	BrokenLinks  []BrokenLink `json:"broken_links"`
	Assets       []Asset      `json:"assets"`
	DiscoveredAt time.Time    `json:"discovered_at"`
}

// BrokenLink is a link found on a page and its check result.
type BrokenLink struct {
	URL        string `json:"url"`
	StatusCode int    `json:"status_code"`
	Error      string `json:"error"`
}

// Report is the crawl report.
type Report struct {
	RootURL     string    `json:"root_url"`
	Depth       uint      `json:"depth"`
	GeneratedAt time.Time `json:"generated_at"`
	Pages       []Page    `json:"pages"`
}

// AssetType identifies the kind of a resource referenced by a page.
type AssetType = assets.Type

// Asset contains the measured result for a resource referenced by a page.
type Asset = assets.Asset

// SEOResult holds basic SEO data of a page.
type SEOResult = seo.Result
