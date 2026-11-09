package assets

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"code/internal/htmlparse"
)

type extractedRef struct {
	URL  string
	Type Type
}

func TestExtract(t *testing.T) {
	html := `
		<html>
			<head>
				<link rel="alternate stylesheet" href="/css/site.css">
				<link rel="icon" href="/favicon.ico">
				<script src="https://cdn.example.test/app.js"></script>
			</head>
			<body>
				<img src="../images/logo.png">
				<img src="../images/logo.png">
				<script src="data:text/javascript,alert(1)"></script>
			</body>
		</html>
	`
	reader := strings.NewReader(html)

	doc, err := htmlparse.Parse(reader)
	require.NoError(t, err)

	pageURL, err := url.Parse("https://example.test/docs/index.html")
	require.NoError(t, err)

	refs := Extract(doc, pageURL)
	got := make([]extractedRef, len(refs))
	for i, ref := range refs {
		refURL := ref.URL.String()
		got[i] = extractedRef{URL: refURL, Type: ref.Type}
	}

	want := []extractedRef{
		{URL: "https://example.test/images/logo.png", Type: TypeImage},
		{URL: "https://cdn.example.test/app.js", Type: TypeScript},
		{URL: "https://example.test/css/site.css", Type: TypeStyle},
	}
	require.Equal(t, want, got)
}
