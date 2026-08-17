package renderers

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/reidransom/jigyll/config"
	"github.com/reidransom/liquid"
	"github.com/stretchr/testify/require"
)

func TestRenderLiquidBeforeSass(t *testing.T) {
	const entrypoint = `{% if site.color_scheme and site.color_scheme != "nil" %}
  {% assign color_scheme = site.color_scheme %}
{% else %}
  {% assign color_scheme = "light" %}
{% endif %}
{% include css/just-the-docs.scss.liquid color_scheme=color_scheme %}`

	for _, tc := range []struct {
		name     string
		site     map[string]interface{}
		expected string
	}{
		{name: "configured color scheme", site: map[string]interface{}{"color_scheme": "dark"}, expected: ".dark{color:red}"},
		{name: "absent color scheme falls back to light", site: map[string]interface{}{}, expected: ".light{color:red}"},
		{name: "nil color scheme falls back to light", site: map[string]interface{}{"color_scheme": "nil"}, expected: ".light{color:red}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			siteDir := t.TempDir()
			includeDir := filepath.Join(siteDir, "_includes", "css")
			require.NoError(t, os.MkdirAll(includeDir, 0o755))
			require.NoError(t, os.WriteFile(
				filepath.Join(includeDir, "just-the-docs.scss.liquid"),
				[]byte(`.{{ include.color_scheme }} { color: red; }`),
				0o644,
			))

			cfg := config.Default()
			cfg.Source = siteDir
			manager, err := New(cfg, Options{})
			require.NoError(t, err)
			t.Cleanup(func() {
				require.NoError(t, os.RemoveAll(manager.sassTempDir))
			})

			var output bytes.Buffer
			err = manager.Render(
				&output,
				[]byte(entrypoint),
				liquid.Bindings{"site": tc.site},
				"assets/css/just-the-docs-default.scss",
				1,
			)
			require.NoError(t, err)
			require.Equal(t, tc.expected, output.String())
			require.NotContains(t, output.String(), "{{")
			require.NotContains(t, output.String(), "{%")
		})
	}
}

func TestRenderScssifyUsesThemeAndSiteSass(t *testing.T) {
	siteDir := t.TempDir()
	themeDir := t.TempDir()

	writePipelineTestFile(t, themeDir, "_sass/support/support.scss", `$theme-color: #112233;`)
	writePipelineTestFile(t, themeDir, "_sass/custom/setup.scss", `$override-color: #ff0000;`)
	writePipelineTestFile(t, siteDir, "_sass/custom/setup.scss", `$override-color: #445566;`)
	writePipelineTestFile(
		t,
		themeDir,
		"_includes/css/just-the-docs.scss.liquid",
		`@import "./support/support";
@import "./custom/setup";
.scheme-{{ include.color_scheme }} {
  color: $theme-color;
  background-color: $override-color;
}`,
	)

	cfg := config.Default()
	cfg.Source = siteDir
	manager, err := New(cfg, Options{ThemeDir: themeDir})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(manager.sassTempDir))
	})

	const entrypoint = `{% assign color_scheme = "light" %}
{% capture scss %}
{% include css/just-the-docs.scss.liquid color_scheme=color_scheme %}
{% endcapture %}
{{ scss | scssify }}`

	var output bytes.Buffer
	err = manager.Render(
		&output,
		[]byte(entrypoint),
		liquid.Bindings{},
		"assets/css/just-the-docs-head-nav.css",
		1,
	)
	require.NoError(t, err)
	require.Contains(t, output.String(), "color: #112233;")
	require.Contains(t, output.String(), "background-color: #445566;")
	require.NotContains(t, output.String(), "#ff0000")
	require.NotContains(t, output.String(), "@import")
	require.NotContains(t, output.String(), "{{")
	require.NotContains(t, output.String(), "{%")
}

func writePipelineTestFile(t *testing.T, root, name, content string) {
	t.Helper()
	filename := filepath.Join(root, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(filename), 0o755))
	require.NoError(t, os.WriteFile(filename, []byte(content), 0o644))
}
