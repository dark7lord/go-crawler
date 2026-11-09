package fetch

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func newTestFetcher(t *testing.T, delay time.Duration) (*Fetcher, *url.URL) {
	t.Helper()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("delay: " + delay.String()))
	})

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	u, err := url.Parse(server.URL)
	require.NoError(t, err)

	ftr := New(Config{
		Client: server.Client(),
		Delay:  delay,
	})

	return ftr, u
}

func newStatusFetcher(t *testing.T, retries uint, statuses ...int) (*Fetcher, *url.URL, *atomic.Int32) {
	t.Helper()
	require.NotEmpty(t, statuses)

	var requests atomic.Int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		index := int(requests.Add(1)) - 1
		if index >= len(statuses) {
			index = len(statuses) - 1
		}
		w.WriteHeader(statuses[index])
	})
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	u, err := url.Parse(server.URL)
	require.NoError(t, err)

	ftr := New(Config{
		Client:  server.Client(),
		Retries: retries,
	})

	return ftr, u, &requests
}

type responseSequence struct {
	responses []*http.Response
}

func (s *responseSequence) RoundTrip(*http.Request) (*http.Response, error) {
	resp := s.responses[0]
	s.responses = s.responses[1:]

	return resp, nil
}

type closeTrackingBody struct {
	closed bool
}

func (*closeTrackingBody) Read([]byte) (int, error) {
	return 0, io.EOF
}

func (b *closeTrackingBody) Close() error {
	b.closed = true

	return nil
}
