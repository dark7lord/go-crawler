package htmlparse

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractLinks(t *testing.T) {
	doc, err := Parse(strings.NewReader(`
		<html>
			<body>
				<a href="/about">About</a>
				<a>No href</a>
				<a href="">Empty</a>
				<a href="#section">Fragment</a>
			</body>
		</html>
	`))
	require.NoError(t, err)

	require.Equal(t, []string{"/about", "", "#section"}, ExtractLinks(doc))
}
