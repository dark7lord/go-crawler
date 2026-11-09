package urlutil

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "scheme-less domain", input: "ya.ru", want: "http://ya.ru"},
		{name: "https passthrough", input: "https://ya.ru/path", want: "https://ya.ru/path"},
		{name: "http passthrough", input: "http://ya.ru/foo", want: "http://ya.ru/foo"},
		{name: "scheme-less localhost with port", input: "localhost:8080/x", want: "http://localhost:8080/x"},
		{name: "scheme-less local ip with port", input: "127.0.0.1:3000", want: "http://127.0.0.1:3000"},
		{name: "path and query preserved", input: "example.com/foo?x=1", want: "http://example.com/foo?x=1"},
		{name: "surrounding spaces are trimmed", input: "  ya.ru  ", want: "http://ya.ru"},
		{name: "unsupported scheme", input: "ftp://x", wantErr: true},
		{name: "javascript scheme", input: "javascript:alert(1)", wantErr: true},
		{name: "empty", input: "", wantErr: true},
		{name: "whitespace only", input: "   ", wantErr: true},
		{name: "no host", input: "http://", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Normalize(tt.input)
			if tt.wantErr {
				require.ErrorIs(t, err, ErrInvalid)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got.String())
		})
	}
}

func TestResolve(t *testing.T) {
	base, err := url.Parse("https://example.test/docs/index.html")
	require.NoError(t, err)

	tests := []struct {
		name string
		href string
		want string
		ok   bool
	}{
		{
			name: "relative path",
			href: "../images/logo.png",
			want: "https://example.test/images/logo.png",
			ok:   true,
		},
		{
			name: "root relative path",
			href: "/about",
			want: "https://example.test/about",
			ok:   true,
		},
		{
			name: "absolute https URL",
			href: "https://cdn.example.test/app.js",
			want: "https://cdn.example.test/app.js",
			ok:   true,
		},
		{
			name: "surrounding spaces are trimmed",
			href: "  /about  ",
			want: "https://example.test/about",
			ok:   true,
		},
		{
			name: "empty",
		},
		{
			name: "whitespace only",
			href: "   ",
		},
		{
			name: "fragment only",
			href: "#section",
		},
		{
			name: "mailto scheme",
			href: "mailto:test@example.test",
		},
		{
			name: "javascript scheme",
			href: "javascript:alert(1)",
		},
		{
			name: "invalid URL",
			href: "%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := Resolve(base, tt.href)

			require.Equal(t, tt.ok, ok)
			if !tt.ok {
				require.Nil(t, got)

				return
			}

			require.Equal(t, tt.want, got.String())
		})
	}
}
