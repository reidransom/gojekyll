package pages

import (
	"fmt"
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

type renderer interface {
	Render() error
}
