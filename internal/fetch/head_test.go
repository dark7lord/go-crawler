package fetch

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFetcherHead(t *testing.T) {
	tests := []struct {
		name        string
		head        func(t *testing.T, w http.ResponseWriter)
		getStatus   int
		wantStatus  int
		wantMethods []string
	}{
		{
			name: "uses head response",
			head: func(_ *testing.T, w http.ResponseWriter) {
				w.WriteHeader(http.StatusNotFound)
			},
			getStatus:   http.StatusOK,
			wantStatus:  http.StatusNotFound,
			wantMethods: []string{http.MethodHead},
		},
		{
			name: "falls back when head is unsupported",
			head: func(_ *testing.T, w http.ResponseWriter) {
				w.WriteHeader(http.StatusMethodNotAllowed)
			},
			getStatus:   http.StatusNotFound,
			wantStatus:  http.StatusNotFound,
			wantMethods: []string{http.MethodHead, http.MethodGet},
		},
		{
			name: "falls back after head transport error",
			head: func(t *testing.T, w http.ResponseWriter) {
				t.Helper()

				conn, _, err := w.(http.Hijacker).Hijack()
				if err != nil {
					t.Errorf("hijack connection: %v", err)

					return
				}
				_ = conn.Close()
			},
			getStatus:   http.StatusOK,
			wantStatus:  http.StatusOK,
			wantMethods: []string{http.MethodHead, http.MethodGet},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var methods []string
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				methods = append(methods, r.Method)
				if r.Method == http.MethodHead {
					tt.head(t, w)

					return
				}
				w.WriteHeader(tt.getStatus)
			})
			server := httptest.NewServer(handler)
			t.Cleanup(server.Close)

			u, err := url.Parse(server.URL)
			require.NoError(t, err)
			ftr := New(Config{Client: server.Client()})

			resp, err := ftr.Head(t.Context(), u)
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, resp.StatusCode)
			require.NoError(t, resp.Body.Close())
			require.Equal(t, tt.wantMethods, methods)
		})
	}
}
