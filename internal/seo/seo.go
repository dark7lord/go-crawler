// Package seo extracts SEO data from HTML documents.
package seo

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Result holds basic SEO data of a page.
type Result struct {
	HasTitle       bool   `json:"has_title"`
	Title          string `json:"title"`
	HasDescription bool   `json:"has_description"`
	Description    string `json:"description"`
	HasH1          bool   `json:"has_h1"`
}

// Extract extracts basic SEO data from a document.
func Extract(doc *goquery.Document) Result {
	title := findTitle(doc)
	desc := doc.Find(`meta[name="description"]`)
	desc = desc.First()

	titleText := strings.TrimSpace(title.Text())
	hasTitle := title.Length() > 0
	hasDescription := desc.Length() > 0
	hasH1 := doc.Find("h1").Length() > 0
	content, _ := desc.Attr("content")

	return Result{
		HasTitle:       hasTitle,
		Title:          titleText,
		HasDescription: hasDescription,
		Description:    strings.TrimSpace(content),
		HasH1:          hasH1,
	}
}

func findTitle(doc *goquery.Document) *goquery.Selection {
	title := doc.Find("head > title")
	if title.Length() > 0 {
		return title.First()
	}

	title = doc.Find("title")
	title = title.FilterFunction(notInSVG)

	return title.First()
}

func notInSVG(_ int, selection *goquery.Selection) bool {
	return selection.Closest("svg").Length() == 0
}
