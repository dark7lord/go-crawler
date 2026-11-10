// Package crawler implements website structure analysis.
package crawler

import "context"

// Analyze crawls a website and returns a report about its structure.
func Analyze(ctx context.Context, opts Options) (data []byte, err error) {
	crw, err := newCrawler(opts)
	if err != nil {
		return nil, err
	}

	crw.crawl(ctx)

	return crw.report()
}
