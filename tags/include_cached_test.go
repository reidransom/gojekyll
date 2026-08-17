package tags

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/reidransom/jigyll/config"
	"github.com/reidransom/liquid"
	"github.com/stretchr/testify/require"
)

const includeCachePluginName = "jekyll-include-cache"

func TestIncludeCachedRegistration(t *testing.T) {
	includeDir := t.TempDir()
	writeIncludeCachedFixture(t, includeDir, "greeting.html", "hello")

	t.Run("configured plugin registers tag", func(t *testing.T) {
		engine := newIncludeCachedEngine(t, []string{includeDir}, true)
		output, err := engine.ParseAndRenderString(`{% include_cached greeting.html %}`, liquid.Bindings{})
		require.NoError(t, err)
		require.Equal(t, "hello", output)
	})

	t.Run("tag is unavailable without plugin", func(t *testing.T) {
		engine := newIncludeCachedEngine(t, []string{includeDir}, false)
		_, err := engine.ParseAndRenderString(`{% include_cached greeting.html %}`, liquid.Bindings{})
		require.ErrorContains(t, err, `undefined tag "include_cached"`)
	})
}

func TestIncludeCachedCachesByEvaluatedArguments(t *testing.T) {
	includeDir := t.TempDir()
	writeIncludeCachedFixture(t, includeDir, "context.html", `{{ ambient }}|{{ include.key }}`)
	engine := newIncludeCachedEngine(t, []string{includeDir}, true)

	const source = `{%- assign ambient = "first" -%}
{%- include_cached context.html key=key_value payload=first_payload -%}
{%- assign ambient = "second" -%}
{%- include_cached context.html payload=equal_payload key="same" -%}
{%- assign ambient = "third" -%}
{%- include_cached context.html key="different" payload=equal_payload -%}`
	bindings := liquid.Bindings{
		"key_value":     "same",
		"first_payload": map[string]interface{}{"items": []interface{}{1, "two"}},
		"equal_payload": map[string]interface{}{"items": []interface{}{1, "two"}},
	}

	output, err := engine.ParseAndRenderString(source, bindings)
	require.NoError(t, err)
	require.Equal(t, "first|samefirst|samethird|different", output)
}

func TestIncludeCachedUsesResolvedPathAndIncludePrecedence(t *testing.T) {
	siteIncludes := t.TempDir()
	themeIncludes := t.TempDir()
	writeIncludeCachedFixture(t, siteIncludes, "shared.html", "site")
	writeIncludeCachedFixture(t, themeIncludes, "shared.html", "theme")
	writeIncludeCachedFixture(t, themeIncludes, "theme-only.html", "theme-only")
	engine := newIncludeCachedEngine(t, []string{siteIncludes, themeIncludes}, true)

	output, err := engine.ParseAndRenderString(
		`{% include_cached shared.html key="same" %}|{% include_cached theme-only.html key="same" %}`,
		liquid.Bindings{},
	)
	require.NoError(t, err)
	require.Equal(t, "site|theme-only", output)
}

func TestIncludeCachedDoesNotCacheErrors(t *testing.T) {
	includeDir := t.TempDir()
	engine := newIncludeCachedEngine(t, []string{includeDir}, true)

	t.Run("missing include matches normal include", func(t *testing.T) {
		_, includeErr := engine.ParseAndRenderString(`{% include missing.html %}`, liquid.Bindings{})
		_, cachedErr := engine.ParseAndRenderString(`{% include_cached missing.html %}`, liquid.Bindings{})
		require.Error(t, includeErr)
		require.Error(t, cachedErr)
		require.ErrorContains(t, includeErr, "missing.html")
		require.ErrorContains(t, cachedErr, "missing.html")
	})

	t.Run("include loop is rejected", func(t *testing.T) {
		writeIncludeCachedFixture(t, includeDir, "loop.html", `{% include_cached loop.html %}`)
		_, err := engine.ParseAndRenderString(`{% include_cached loop.html %}`, liquid.Bindings{})
		require.ErrorContains(t, err, "include loop detected")
	})

	t.Run("failed render can succeed later", func(t *testing.T) {
		writeIncludeCachedFixture(t, includeDir, "repairable.html", `{% no_such_tag %}`)
		_, err := engine.ParseAndRenderString(`{% include_cached repairable.html key="same" %}`, liquid.Bindings{})
		require.Error(t, err)

		writeIncludeCachedFixture(t, includeDir, "repairable.html", "fixed")
		output, err := engine.ParseAndRenderString(`{% include_cached repairable.html key="same" %}`, liquid.Bindings{})
		require.NoError(t, err)
		require.Equal(t, "fixed", output)
	})
}

func TestIncludeCachedCacheLifetime(t *testing.T) {
	includeDir := t.TempDir()
	writeIncludeCachedFixture(t, includeDir, "ambient.html", `{{ ambient }}`)

	firstEngine := newIncludeCachedEngine(t, []string{includeDir}, true)
	output, err := firstEngine.ParseAndRenderString(
		`{% include_cached ambient.html %}{% include_cached ambient.html %}`,
		liquid.Bindings{"ambient": "first"},
	)
	require.NoError(t, err)
	require.Equal(t, "firstfirst", output)

	secondEngine := newIncludeCachedEngine(t, []string{includeDir}, true)
	output, err = secondEngine.ParseAndRenderString(
		`{% include_cached ambient.html %}`,
		liquid.Bindings{"ambient": "second"},
	)
	require.NoError(t, err)
	require.Equal(t, "second", output)
}

func TestIncludeCacheStoresEmptyOutput(t *testing.T) {
	cache := newIncludeCache()
	params := map[string]interface{}{"key": "value"}

	require.Equal(t, "", cache.store("empty.html", params, ""))
	output, ok := cache.lookup("empty.html", params)
	require.True(t, ok)
	require.Equal(t, "", output)
}

func newIncludeCachedEngine(t *testing.T, includeDirs []string, enabled bool) *liquid.Engine {
	t.Helper()
	cfg := config.Default()
	if enabled {
		cfg.Plugins = []string{includeCachePluginName}
	}
	engine := liquid.NewEngine()
	AddJekyllTags(engine, &cfg, includeDirs, func(string) (string, bool) { return "", false })
	return engine
}

func writeIncludeCachedFixture(t *testing.T, root, name, content string) {
	t.Helper()
	filename := filepath.Join(root, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(filename), 0o755))
	require.NoError(t, os.WriteFile(filename, []byte(content), 0o644))
}
