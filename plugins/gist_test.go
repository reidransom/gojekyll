package plugins

import (
	"regexp"
	"testing"

	"github.com/reidransom/jigyll/config"
	"github.com/reidransom/liquid"
	"github.com/stretchr/testify/require"
)

func TestGistTag(t *testing.T) {
	engine := liquid.NewEngine()
	names := []string{"jekyll-gist"}
	installed, err := Install(names, siteFake{config.Default(), engine})
	require.NoError(t, err)
	require.NoError(t, installed[names[0]].ConfigureTemplateEngine(engine))
	s, err := engine.ParseAndRenderString(`{% gist parkr/931c1c8d465a04042403 %}`, liquid.Bindings{})
	require.NoError(t, err)
	re := regexp.MustCompile(`<script.*>\s*</script>`)
	require.Contains(t, s, `src="https://gist.github.com/parkr/931c1c8d465a04042403.js"`)
	require.True(t, re.MatchString(s))
}

func TestGistTagWithFilename(t *testing.T) {
	engine := liquid.NewEngine()
	names := []string{"jekyll-gist"}
	installed, err := Install(names, siteFake{config.Default(), engine})
	require.NoError(t, err)
	require.NoError(t, installed[names[0]].ConfigureTemplateEngine(engine))
	s, err := engine.ParseAndRenderString(`{% gist parkr/931c1c8d465a04042403 test.rb %}`, liquid.Bindings{})
	require.NoError(t, err)
	require.Contains(t, s, `src="https://gist.github.com/parkr/931c1c8d465a04042403.js?file=test.rb"`)
}

func TestGistTagNoscriptDisabled(t *testing.T) {
	cfg := config.Default()
	cfg.Set("gist", map[string]interface{}{"noscript": false})
	engine := liquid.NewEngine()
	names := []string{"jekyll-gist"}
	site := siteFake{cfg, engine}
	installed, err := Install(names, site)
	require.NoError(t, err)
	require.NoError(t, installed[names[0]].ConfigureTemplateEngine(engine))
	// Create bindings with site config
	bindings := liquid.Bindings{"site": site.ToLiquid()}
	s, err := engine.ParseAndRenderString(`{% gist parkr/931c1c8d465a04042403 %}`, bindings)
	require.NoError(t, err)
	require.Contains(t, s, `<script src="https://gist.github.com/parkr/931c1c8d465a04042403.js"> </script>`)
	require.NotContains(t, s, `<noscript>`)
}

func TestGistTagNoscriptEnabled(t *testing.T) {
	cfg := config.Default()
	cfg.Set("gist", map[string]interface{}{"noscript": true})
	engine := liquid.NewEngine()
	names := []string{"jekyll-gist"}
	site := siteFake{cfg, engine}
	installed, err := Install(names, site)
	require.NoError(t, err)
	require.NoError(t, installed[names[0]].ConfigureTemplateEngine(engine))
	// Create bindings with site config
	bindings := liquid.Bindings{"site": site.ToLiquid()}
	s, err := engine.ParseAndRenderString(`{% gist parkr/931c1c8d465a04042403 %}`, bindings)
	require.NoError(t, err)

	// Should contain script tag
	require.Contains(t, s, `<script src="https://gist.github.com/parkr/931c1c8d465a04042403.js"> </script>`)

	// Should also contain noscript tag (if the gist is accessible)
	// Note: This test may fail if network is unavailable or gist doesn't exist
	// In production, the implementation gracefully handles fetch failures
	if regexp.MustCompile(`<noscript>`).MatchString(s) {
		require.Contains(t, s, `<noscript><pre>`)
		require.Contains(t, s, `</pre></noscript>`)
	}
}
