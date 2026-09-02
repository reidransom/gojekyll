package plugins

import (
	"testing"

	"github.com/reidransom/jigyll/config"
	"github.com/reidransom/liquid"
	"github.com/stretchr/testify/require"
)

func TestFeedPluginKeepsSiteStateSeparate(t *testing.T) {
	englishConfig := config.Default()
	englishConfig.AbsoluteURL = "https://english.example"
	englishConfig.Set("title", "English")
	englishSite := &siteFake{englishConfig, liquid.NewEngine()}

	germanConfig := config.Default()
	germanConfig.AbsoluteURL = "https://deutsch.example"
	germanConfig.Set("title", "Deutsch")
	germanSite := &siteFake{germanConfig, liquid.NewEngine()}

	names := []string{"jekyll-feed"}
	englishPlugins, err := Install(names, englishSite)
	require.NoError(t, err)
	germanPlugins, err := Install(names, germanSite)
	require.NoError(t, err)

	englishPlugin := englishPlugins["jekyll-feed"].(*jekyllFeedPlugin)
	germanPlugin := germanPlugins["jekyll-feed"].(*jekyllFeedPlugin)
	require.NotSame(t, englishPlugin, germanPlugin)

	// Initialize both sites before either template renders.
	require.NoError(t, englishPlugin.ConfigureTemplateEngine(englishSite.e))
	require.NoError(t, germanPlugin.ConfigureTemplateEngine(germanSite.e))

	englishOutput, err := englishSite.e.ParseAndRenderString(`{% feed_meta %}`, liquid.Bindings{})
	require.NoError(t, err)
	require.Contains(t, englishOutput, `href="https://english.example/feed.xml" title="English"`)

	germanOutput, err := germanSite.e.ParseAndRenderString(`{% feed_meta %}`, liquid.Bindings{})
	require.NoError(t, err)
	require.Contains(t, germanOutput, `href="https://deutsch.example/feed.xml" title="Deutsch"`)
}
