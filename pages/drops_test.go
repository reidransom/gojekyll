package pages

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/reidransom/jigyll/config"
	"github.com/reidransom/liquid"
	"github.com/reidransom/liquid/tags"
	"github.com/stretchr/testify/require"
)

func TestStaticFile_ToLiquid(t *testing.T) {
	site := siteFake{t, config.Default()}
	page, err := NewFile(site, "testdata/static.html", "static.html", func(bool) FrontMatter { return FrontMatter{} })
	require.NoError(t, err)
	drop := page.(liquid.Drop).ToLiquid().(tags.IterationKeyedMap)

	require.Equal(t, "static", drop["basename"])
	require.Equal(t, "static.html", drop["name"])
	require.Equal(t, "/static.html", drop["path"])
	require.Equal(t, ".html", drop["extname"])
	require.IsType(t, time.Now(), drop["modified_time"])
}

func TestStaticFile_ToLiquid_defaults(t *testing.T) {
	site := siteFake{t, config.Default()}
	page, err := NewFile(site, "testdata/static.html", "static.html", func(bool) FrontMatter {
		return FrontMatter{"sitemap": false, "image": true, "path": "should-be-shadowed", "name": "should-be-shadowed"}
	})
	require.NoError(t, err)
	drop := page.(liquid.Drop).ToLiquid().(tags.IterationKeyedMap)

	// Default front matter surfaces through the drop.
	require.Equal(t, false, drop["sitemap"])
	require.Equal(t, true, drop["image"])
	// Fixed metadata keys are not overridable by defaults.
	require.Equal(t, "/static.html", drop["path"])
	require.Equal(t, "static.html", drop["name"])
}

func TestPage_ToLiquid_excerpt(t *testing.T) {
	site := siteFake{t, config.Default()}
	p, err := NewFile(site, "testdata/excerpt.md", "excerpt.md", func(bool) FrontMatter { return FrontMatter{} })
	require.NoError(t, err)

	t.Run("before render", func(t *testing.T) {
		drop := p.(liquid.Drop).ToLiquid()
		excerpt := drop.(tags.IterationKeyedMap)["excerpt"]
		require.Equal(t, "First line.", fmt.Sprintf("%s", excerpt))
	})

	t.Run("after render", func(t *testing.T) {
		require.NoError(t, p.(renderer).Render())
		drop := p.(liquid.Drop).ToLiquid()
		excerpt := drop.(tags.IterationKeyedMap)["excerpt"]
		require.Equal(t, "rendered: First line.", fmt.Sprintf("%s", excerpt))
	})
}

func TestPage_ToLiquid_name(t *testing.T) {
	site := siteFake{t, config.Default()}
	p, err := NewFile(site, "testdata/excerpt.md", "excerpt.md", func(bool) FrontMatter { return FrontMatter{} })
	require.NoError(t, err)

	drop := p.(liquid.Drop).ToLiquid().(tags.IterationKeyedMap)
	require.Equal(t, "excerpt.md", drop["name"])
	require.Equal(t, "testdata/excerpt.md", drop["path"])
}

func TestPage_ToLiquid_date(t *testing.T) {
	modified := time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC)
	explicit := time.Date(2023, time.February, 3, 4, 5, 6, 0, time.UTC)

	t.Run("omits an undated page date", func(t *testing.T) {
		drop, _ := newPageDrop(t, "---\n---\ncontent\n", "page.md", FrontMatter{}, modified)
		_, hasDate := drop["date"]
		_, hasModifiedTime := drop["modified_time"]

		require.False(t, hasDate)
		require.False(t, hasModifiedTime)
	})

	t.Run("retains an explicit page date", func(t *testing.T) {
		drop, _ := newPageDrop(t, "---\ndate: 2023-02-03 04:05:06 +00:00\n---\ncontent\n", "page.md", FrontMatter{}, modified)

		require.True(t, explicit.Equal(drop["date"].(time.Time)))
	})
}

func TestPage_ToLiquid_postDate(t *testing.T) {
	modified := time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC)
	explicit := time.Date(2023, time.February, 3, 4, 5, 6, 0, time.UTC)
	post := FrontMatter{"collection": "posts"}

	t.Run("retains a post front matter date", func(t *testing.T) {
		drop, _ := newPageDrop(t, "---\ndate: 2023-02-03 04:05:06 +00:00\n---\ncontent\n", "2023-02-03-post.md", post, modified)

		require.True(t, explicit.Equal(drop["date"].(time.Time)))
	})

	t.Run("falls back to a dateless post modification time", func(t *testing.T) {
		drop, fileModified := newPageDrop(t, "---\n---\ncontent\n", "post.md", post, modified)

		require.Equal(t, fileModified, drop["date"])
	})
}

func newPageDrop(t *testing.T, contents, relPath string, defaults FrontMatter, modified time.Time) (tags.IterationKeyedMap, time.Time) {
	t.Helper()

	filename := filepath.Join(t.TempDir(), relPath)
	require.NoError(t, os.WriteFile(filename, []byte(contents), 0o644))
	require.NoError(t, os.Chtimes(filename, modified, modified))
	info, err := os.Stat(filename)
	require.NoError(t, err)

	document, err := NewFile(siteFake{t, config.Default()}, filename, relPath, func(bool) FrontMatter {
		return defaults
	})
	require.NoError(t, err)
	return document.(liquid.Drop).ToLiquid().(tags.IterationKeyedMap), info.ModTime()
}

type renderer interface {
	Render() error
}
