package assets

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"

	"code/internal/fetch"
)

// Loader loads and caches assets shared by the whole crawl.
type Loader struct {
	fetcher *fetch.Fetcher
	mu      sync.Mutex
	cache   map[string]*cacheEntry
}

type cacheEntry struct {
	ready chan struct{}
	asset *Asset
}

// NewLoader creates a Loader that fetches assets with fetcher.
func NewLoader(fetcher *fetch.Fetcher) *Loader {
	return &Loader{
		fetcher: fetcher,
		cache:   map[string]*cacheEntry{},
	}
}

// Load fetches all refs, sharing concurrent loads between callers.
func (loader *Loader) Load(ctx context.Context, refs []Ref) []Asset {
	assets := make([]Asset, 0, len(refs))

	for _, ref := range refs {
		if asset := loader.load(ctx, ref); asset != nil {
			assets = append(assets, *asset)
		}
	}

	return assets
}

func (loader *Loader) load(ctx context.Context, ref Ref) *Asset {
	key := ref.URL.String()

	for ctx.Err() == nil {
		entry, isLoadOwner := loader.beginLoad(key)
		if isLoadOwner {
			asset := loader.loadOnce(ctx, ref)
			loader.finishLoad(key, entry, asset)

			return asset
		}

		cached, err := waitForLoad(ctx, entry)
		if err != nil {
			return nil
		}
		if cached == nil {
			continue
		}

		cached.Type = ref.Type

		return cached
	}

	return nil
}

// beginLoad returns the cache entry and whether this caller owns its load.
func (loader *Loader) beginLoad(key string) (*cacheEntry, bool) {
	loader.mu.Lock()
	defer loader.mu.Unlock()

	if entry, exists := loader.cache[key]; exists {
		return entry, false
	}

	entry := &cacheEntry{ready: make(chan struct{})}
	loader.cache[key] = entry

	return entry, true
}

// finishLoad publishes the result and wakes waiters; cancelled loads are removed.
func (loader *Loader) finishLoad(key string, entry *cacheEntry, asset *Asset) {
	loader.mu.Lock()
	defer loader.mu.Unlock()

	entry.asset = asset
	if asset == nil {
		delete(loader.cache, key)
	}
	close(entry.ready)
}

// waitForLoad waits for the shared load or for this caller's context to be cancelled.
func waitForLoad(ctx context.Context, entry *cacheEntry) (*Asset, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-entry.ready:
		if entry.asset == nil {
			return nil, nil
		}

		asset := *entry.asset

		return &asset, nil
	}
}

func (loader *Loader) loadOnce(ctx context.Context, ref Ref) *Asset {
	asset := &Asset{URL: ref.URL.String(), Type: ref.Type}

	resp, err := loader.fetcher.Get(ctx, ref.URL)
	if err != nil {
		if ctx.Err() != nil {
			return nil
		}

		asset.Error = err.Error()

		return asset
	}
	defer func() { _ = resp.Body.Close() }()

	asset.StatusCode = resp.StatusCode
	if resp.StatusCode >= http.StatusBadRequest {
		statusText := http.StatusText(resp.StatusCode)
		asset.Error = fmt.Sprintf("HTTP %d %s", resp.StatusCode, statusText)
	}

	size, err := readSize(resp)
	if err != nil {
		if ctx.Err() != nil {
			return nil
		}

		if asset.Error != "" {
			asset.Error += "; "
		}
		asset.Error += err.Error()
	} else {
		asset.SizeBytes = size
	}

	return asset
}

func readSize(resp *http.Response) (int64, error) {
	if resp.ContentLength >= 0 {
		return resp.ContentLength, nil
	}

	n, err := io.Copy(io.Discard, resp.Body)
	if err != nil {
		return 0, fmt.Errorf("read response body: %w", err)
	}

	return n, nil
}
