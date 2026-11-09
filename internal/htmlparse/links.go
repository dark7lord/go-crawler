package htmlparse

import "github.com/PuerkitoBio/goquery"

// ExtractLinks returns href values of links in a document.
func ExtractLinks(doc *goquery.Document) []string {
	links := doc.Find("a[href]")

	return links.Map(func(_ int, link *goquery.Selection) string {
		return link.AttrOr("href", "")
	})
}
