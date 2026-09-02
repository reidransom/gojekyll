package plugins

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderLocalizedSitemapSortsEscapesAndIncludesAlternates(t *testing.T) {
	output := RenderLocalizedSitemap([]SitemapEntry{
		{
			URL:          "https://example.test/z?x=1&y=2",
			LastModified: "2026-01-02T03:04:05+00:00",
			Alternates: []SitemapAlternate{
				{Language: "fr", URL: "https://example.test/fr?x=1&y=2"},
				{Language: "en", URL: "https://example.test/en?x=1&y=2", XDefault: true},
			},
		},
		{URL: "https://example.test/a"},
	})

	firstEntry := strings.Index(output, "<loc>https://example.test/a</loc>")
	secondEntry := strings.Index(output, "<loc>https://example.test/z?x=1&amp;y=2</loc>")
	require.NotEqual(t, -1, firstEntry)
	require.NotEqual(t, -1, secondEntry)
	require.Less(t, firstEntry, secondEntry)
	require.Contains(t, output, "<lastmod>2026-01-02T03:04:05+00:00</lastmod>")
	englishAlternate := strings.Index(output, `hreflang="en" href="https://example.test/en?x=1&amp;y=2"`)
	frenchAlternate := strings.Index(output, `hreflang="fr" href="https://example.test/fr?x=1&amp;y=2"`)
	require.NotEqual(t, -1, englishAlternate)
	require.NotEqual(t, -1, frenchAlternate)
	require.Less(t, englishAlternate, frenchAlternate)
	require.Contains(t, output, `hreflang="x-default" href="https://example.test/en?x=1&amp;y=2"`)
}
