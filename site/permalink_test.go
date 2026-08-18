package site

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/reidransom/jigyll/config"
	"github.com/stretchr/testify/require"
)

func TestGlobalPrettyPageRoutesAndDestinations(t *testing.T) {
	source := t.TempDir()
	writePrettyPermalinkSiteFile(t, source, "_config.yml", "permalink: pretty\n")
	writePrettyPermalinkSiteFile(t, source, "docs/configuration.md", "---\n---\n{{ page.url }}\n")
	writePrettyPermalinkSiteFile(t, source, "docs/index.md", "---\n---\nDocs\n")
	writePrettyPermalinkSiteFile(t, source, "docs/contacts.htm", "---\n---\nContacts\n")
	writePrettyPermalinkSiteFile(t, source, "sitemap.xml", "---\n---\n<sitemap/>\n")
	writePrettyPermalinkSiteFile(t, source, "legacy.md", "---\npermalink: /legacy.html\n---\nLegacy\n")
	writePrettyPermalinkSiteFile(t, source, "asset.html", "static asset\n")

	s, err := FromDirectory(source, config.Flags{})
	require.NoError(t, err)
	require.NoError(t, s.Read())

	require.Contains(t, s.Routes, "/docs/configuration/")
	require.NotContains(t, s.Routes, "/docs/configuration.html")
	require.Contains(t, s.Routes, "/docs/contacts/")
	require.Contains(t, s.Routes, "/sitemap.xml")
	require.Contains(t, s.Routes, "/legacy.html")
	require.Contains(t, s.Routes, "/asset.html")
	_, found := s.URLPage("/docs/configuration/")
	require.True(t, found)
	_, found = s.URLPage("/docs/configuration.html")
	require.False(t, found)

	count, err := s.Write()
	require.NoError(t, err)
	require.Equal(t, 6, count)

	dest := s.DestDir()
	require.FileExists(t, filepath.Join(dest, "docs/configuration/index.html"))
	require.NoFileExists(t, filepath.Join(dest, "docs/configuration.html"))
	require.FileExists(t, filepath.Join(dest, "docs/index.html"))
	require.FileExists(t, filepath.Join(dest, "docs/contacts/index.htm"))
	require.FileExists(t, filepath.Join(dest, "sitemap.xml"))
	require.FileExists(t, filepath.Join(dest, "legacy.html"))
	require.FileExists(t, filepath.Join(dest, "asset.html"))

	page, err := os.ReadFile(filepath.Join(dest, "docs/configuration/index.html"))
	require.NoError(t, err)
	require.Contains(t, string(page), "/docs/configuration/")
}

func writePrettyPermalinkSiteFile(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
