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
