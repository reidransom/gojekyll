package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/reidransom/jigyll/config"
	"github.com/stretchr/testify/require"
)

func TestSiteHtmlPagesSupportJustTheDocsNavigation(t *testing.T) {
	sourceDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(sourceDir, "_includes"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(sourceDir, "_layouts"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(sourceDir, "docs"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "_config.yml"), []byte("plugins:\n  - jekyll-include-cache\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "_includes", "navigation.html"), []byte(`{% include pages.html pages=site.html_pages %}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "_includes", "pages.html"), []byte(`{% assign nav_parenthood = include.pages | where_exp: "item", "item.title != nil" | group_by: "parent" %}{% assign nav_top_nodes = nav_parenthood | where_exp: "item", "item.name == ''" | map: "items" | first %}{% include sorted.html pages=nav_top_nodes %}{{ nav_sorted | size }}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "_includes", "sorted.html"), []byte(`{% assign nav_sorted = include.pages %}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "_layouts", "default.html"), []byte(`{% include_cached navigation.html %}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "index.md"), []byte("---\nlayout: default\ntitle: Home\n---\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "docs", "getting-started.md"), []byte("---\nlayout: default\ntitle: Getting started\n---\n"), 0o644))

	site, err := FromDirectory(sourceDir, config.Flags{})
	require.NoError(t, err)
	require.NoError(t, site.Read())

	require.Equal(t, "2", strings.TrimSpace(renderRoute(t, site, "/")))
}

func TestJustTheDocsNavigation(t *testing.T) {
	site, err := FromDirectory("testdata/just-the-docs-navigation", config.Flags{})
	require.NoError(t, err)
	require.NoError(t, site.Read())

	sidebar := renderRoute(t, site, "/")
	require.Contains(t, sidebar, `<ul class="nav-list">`)
	require.Contains(t, sidebar, `href="/getting-started/"`)
	require.Contains(t, sidebar, `href="/docs/ui-components/"`)
	require.Contains(t, sidebar, `href="/docs/ui-components/code/"`)

	grandchild := renderRoute(t, site, "/docs/ui-components/code/line-numbers/")
	uiComponentsIndex := strings.Index(grandchild, `href="/docs/ui-components/">UI Components</a>`)
	codeIndex := strings.Index(grandchild, `href="/docs/ui-components/code/">Code</a>`)
	lineNumbersIndex := strings.Index(grandchild, `<span>Line Numbers</span>`)
	require.GreaterOrEqual(t, uiComponentsIndex, 0)
	require.Greater(t, codeIndex, uiComponentsIndex)
	require.Greater(t, lineNumbersIndex, codeIndex)

	parent := renderRoute(t, site, "/docs/ui-components/")
	require.Contains(t, parent, `<h2>Children</h2>`)
	require.Contains(t, parent, `href="/docs/ui-components/api/">API</a> - API reference`)
	require.Contains(t, parent, `href="/docs/ui-components/code/">Code</a>`)
}
