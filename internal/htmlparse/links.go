package htmlparse

import "github.com/PuerkitoBio/goquery"

// ExtractLinks returns href values of links in a document.
func ExtractLinks(doc *goquery.Document) []string {
	return doc.Find("a[href]").Map(func(_ int, s *goquery.Selection) string {
		return s.AttrOr("href", "")
	})
}
