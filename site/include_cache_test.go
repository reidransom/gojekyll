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

func TestIncludeSiteBuildPreservesConcurrentLayoutBindings(t *testing.T) {
	sourceDir := t.TempDir()
	writeIncludeCacheSiteFile(t, sourceDir, "_config.yml", "destination: _site\n")
	writeIncludeCacheSiteFile(t, sourceDir, "_data/navigation.yml", "label: catalogue\n")
	writeIncludeCacheSiteFile(t, sourceDir, "_layouts/default.html", `[{% include navigation.html label=page.nav_label %}]{{ content }}`)
	writeIncludeCacheSiteFile(t, sourceDir, "_includes/navigation.html", `page={{ page.title }};locale={{ page.locale }};site={{ site.data.navigation.label }};argument={{ include.label }}`)

	pages := []struct {
		filename string
		title    string
		locale   string
		label    string
	}{
		{"alpha.md", "Alpha", "en", "alpha-argument"},
		{"bravo.md", "Bravo", "fr", "bravo-argument"},
		{"charlie.md", "Charlie", "de", "charlie-argument"},
		{"delta.md", "Delta", "es", "delta-argument"},
	}
	for _, page := range pages {
		writeIncludeCacheSiteFile(t, sourceDir, page.filename, "---\nlayout: default\ntitle: "+page.title+"\nlocale: "+page.locale+"\nnav_label: "+page.label+"\npermalink: /"+page.filename[:len(page.filename)-3]+"/\n---\n"+page.title+" body")
	}

	s := buildIncludeCacheSite(t, sourceDir, config.Flags{})
	for _, page := range pages {
		output := readIncludeCacheSiteOutput(t, sourceDir, page.filename[:len(page.filename)-3], "index.html")
		require.Contains(t, output, "page="+page.title+";locale="+page.locale+";site=catalogue;argument="+page.label)
	}
	require.NotNil(t, s.RendererManager())
}

func TestIncludeSiteBuildRetainsResolutionFallbacksAndDiagnostics(t *testing.T) {
	sourceDir := t.TempDir()
	writeIncludeCacheSiteFile(t, sourceDir, "_config.yml", "theme: test-theme\n")
	writeIncludeCacheSiteFile(t, sourceDir, "_layouts/default.html", `{% include shared.html %}|{% include fallback.html %}|{% include repairable.html %}|{{ content }}`)
	writeIncludeCacheSiteFile(t, sourceDir, "_includes/shared.html", "site")
	writeIncludeCacheSiteFile(t, sourceDir, "_includes/repairable.html", "{% no_such_tag %}")
	writeIncludeCacheSiteFile(t, sourceDir, "_theme/test-theme/_includes/shared.html", "theme")
	writeIncludeCacheSiteFile(t, sourceDir, "_theme/test-theme/_includes/fallback.html", "theme fallback")
	writeIncludeCacheSiteFile(t, sourceDir, "_theme/test-theme/_includes/repairable.html", "recovered")
	writeIncludeCacheSiteFile(t, sourceDir, "index.md", "---\nlayout: default\n---\npage body")

	buildIncludeCacheSite(t, sourceDir, config.Flags{})
	require.Contains(t, readIncludeCacheSiteOutput(t, sourceDir, "index.html"), "site|theme fallback|recovered|")

	writeIncludeCacheSiteFile(t, sourceDir, "_layouts/default.html", "{% include missing.html %}")
	s, err := FromDirectory(sourceDir, config.Flags{})
	require.NoError(t, err)
	require.NoError(t, s.Read())
	_, err = s.Write()
	require.ErrorContains(t, err, "missing.html")
}

func TestIncludeCachedSiteBuildRetainsArgumentResults(t *testing.T) {
	sourceDir := t.TempDir()
	writeIncludeCacheSiteFile(t, sourceDir, "_config.yml", `plugins:
  - jekyll-include-cache
`)
	writeIncludeCacheSiteFile(t, sourceDir, "_layouts/default.html", `{% include_cached marker.html key=page.cache_key %}|{{ content }}`)
	writeIncludeCacheSiteFile(t, sourceDir, "_includes/marker.html", `cached={{ include.key }}`)
	writeIncludeCacheSiteFile(t, sourceDir, "first.md", "---\nlayout: default\ncache_key: first\npermalink: /first/\n---\nfirst body")
	writeIncludeCacheSiteFile(t, sourceDir, "second.md", "---\nlayout: default\ncache_key: second\npermalink: /second/\n---\nsecond body")

	buildIncludeCacheSite(t, sourceDir, config.Flags{})
	require.Contains(t, readIncludeCacheSiteOutput(t, sourceDir, "first", "index.html"), "cached=first|")
	require.Contains(t, readIncludeCacheSiteOutput(t, sourceDir, "second", "index.html"), "cached=second|")
}

func TestIncludeSiteReloadedAfterSourceIncludeChanges(t *testing.T) {
	changes := []struct {
		name    string
		setup   func(t *testing.T, root string)
		change  func(t *testing.T, root string)
		paths   []string
		initial string
		output  string
		err     string
	}{
		{
			name: "edit",
			setup: func(t *testing.T, root string) {
				writeIncludeCacheSiteFile(t, root, "_includes/navigation.html", "before")
			},
			change: func(t *testing.T, root string) {
				writeIncludeCacheSiteFile(t, root, "_includes/navigation.html", "after")
			},
			paths:   []string{"_includes/navigation.html"},
			initial: "before",
			output:  "after",
		},
		{
			name: "create site override",
			setup: func(t *testing.T, root string) {
				writeIncludeCacheSiteFile(t, root, "_theme/test-theme/_includes/navigation.html", "theme")
			},
			change: func(t *testing.T, root string) {
				writeIncludeCacheSiteFile(t, root, "_includes/navigation.html", "site")
			},
			paths:   []string{"_includes/navigation.html"},
			initial: "theme",
			output:  "site",
		},
		{
			name: "remove site override",
			setup: func(t *testing.T, root string) {
				writeIncludeCacheSiteFile(t, root, "_includes/navigation.html", "site")
				writeIncludeCacheSiteFile(t, root, "_theme/test-theme/_includes/navigation.html", "theme")
			},
			change: func(t *testing.T, root string) {
				require.NoError(t, os.Remove(filepath.Join(root, "_includes", "navigation.html")))
			},
			paths:   []string{"_includes/navigation.html"},
			initial: "site",
			output:  "theme",
		},
		{
			name: "rename missing include",
			setup: func(t *testing.T, root string) {
				writeIncludeCacheSiteFile(t, root, "_includes/navigation.html", "site")
			},
			change: func(t *testing.T, root string) {
				require.NoError(t, os.Rename(filepath.Join(root, "_includes", "navigation.html"), filepath.Join(root, "_includes", "moved.html")))
			},
			paths:   []string{"_includes/navigation.html", "_includes/moved.html"},
			initial: "site",
			err:     "navigation.html",
		},
	}

	for _, incremental := range []bool{false, true} {
		incremental := incremental
		mode := "non-incremental"
		if incremental {
			mode = "incremental"
		}
		t.Run(mode, func(t *testing.T) {
			for _, change := range changes {
				change := change
				t.Run(change.name, func(t *testing.T) {
					sourceDir := t.TempDir()
					writeIncludeCacheSiteFile(t, sourceDir, "_config.yml", "theme: test-theme\n")
					writeIncludeCacheSiteFile(t, sourceDir, "_layouts/default.html", `{% include navigation.html %}|{{ content }}`)
					writeIncludeCacheSiteFile(t, sourceDir, "index.md", "---\nlayout: default\n---\npage body")
					change.setup(t, sourceDir)

					s := buildIncludeCacheSite(t, sourceDir, config.Flags{Incremental: &incremental})
					require.Contains(t, readIncludeCacheSiteOutput(t, sourceDir, "index.html"), change.initial)
					change.change(t, sourceDir)

					reloaded, _, err := s.rebuild(change.paths)
					require.NotNil(t, reloaded)
					require.NotSame(t, s, reloaded)
					if change.err != "" {
						require.ErrorContains(t, err, change.err)
						return
					}
					require.NoError(t, err)
					require.Contains(t, readIncludeCacheSiteOutput(t, sourceDir, "index.html"), change.output)
				})
			}
		})
	}
}

func buildIncludeCacheSite(t *testing.T, sourceDir string, flags config.Flags) *Site {
	t.Helper()
	s, err := FromDirectory(sourceDir, flags)
	require.NoError(t, err)
	require.NoError(t, s.Read())
	count, err := s.Write()
	require.NoError(t, err)
	require.Positive(t, count)
	return s
}

func readIncludeCacheSiteOutput(t *testing.T, root string, names ...string) string {
	t.Helper()
	output, err := os.ReadFile(filepath.Join(append([]string{root, "_site"}, names...)...))
	require.NoError(t, err)
	return string(output)
}

func writeIncludeCacheSiteFile(t *testing.T, root, name, content string) {
	t.Helper()
	filename := filepath.Join(root, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(filename), 0o755))
	require.NoError(t, os.WriteFile(filename, []byte(content), 0o644))
}
