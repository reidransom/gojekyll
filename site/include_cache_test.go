package site

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/reidransom/jigyll/config"
	"github.com/stretchr/testify/require"
)

func TestIncludeCachedSiteBuild(t *testing.T) {
	sourceDir := t.TempDir()
	writeIncludeCacheSiteFile(t, sourceDir, "_config.yml", `theme: test-theme
plugins:
  - jekyll-include-cache
`)
	writeIncludeCacheSiteFile(t, sourceDir, "_layouts/default.html", `site:{% include_cached nav.html %}|theme:{% include_cached theme-only.html %}|{{ content }}`)
	writeIncludeCacheSiteFile(t, sourceDir, "_includes/nav.html", "site navigation")
	writeIncludeCacheSiteFile(t, sourceDir, "_theme/test-theme/_includes/nav.html", "theme navigation")
	writeIncludeCacheSiteFile(t, sourceDir, "_theme/test-theme/_includes/theme-only.html", "theme fallback")
	writeIncludeCacheSiteFile(t, sourceDir, "index.md", "---\nlayout: default\n---\npage body")

	s, err := FromDirectory(sourceDir, config.Flags{})
	require.NoError(t, err)
	require.NoError(t, s.Read())
	count, err := s.Write()
	require.NoError(t, err)
	require.Positive(t, count)

	output, err := os.ReadFile(filepath.Join(sourceDir, "_site", "index.html"))
	require.NoError(t, err)
	require.Contains(t, string(output), "site:site navigation")
	require.Contains(t, string(output), "theme:theme fallback")
	require.NotContains(t, string(output), "theme navigation")
	require.Contains(t, string(output), "page body")
}

func writeIncludeCacheSiteFile(t *testing.T, root, name, content string) {
	t.Helper()
	filename := filepath.Join(root, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(filename), 0o755))
	require.NoError(t, os.WriteFile(filename, []byte(content), 0o644))
}
