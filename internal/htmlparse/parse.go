// Package htmlparse extracts structured data from HTML documents.
package htmlparse

import (
	"io"

	"github.com/PuerkitoBio/goquery"
)

// Parse reads an HTML document.
func Parse(r io.Reader) (*goquery.Document, error) {
	return goquery.NewDocumentFromReader(r)
}
