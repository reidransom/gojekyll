package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfig_SourceDir(t *testing.T) {
	c := Default()
	require.True(t, strings.HasPrefix(c.SourceDir(), "/"))
}
func TestDefaultConfig(t *testing.T) {
	c := Default()
	require.Equal(t, ".", c.Source)
	require.Equal(t, "./_site", c.Destination)
	require.Equal(t, "_layouts", c.LayoutsDir)
}

func TestConfig_Map(t *testing.T) {
	c := Default()
	require.NoError(t, Unmarshal([]byte("kramdown:\n  toc_levels: \"2..3\""), &c))
	m, ok := c.Map("kramdown")
	require.True(t, ok)
	require.Equal(t, "2..3", m["toc_levels"])

	_, ok = c.Map("missing")
	require.False(t, ok)

	c = Default()
	require.NoError(t, Unmarshal([]byte(`title: scalar`), &c))
	_, ok = c.Map("title")
	require.False(t, ok)
}

func TestConfig_Plugins(t *testing.T) {
	c := Default()
	require.NoError(t, Unmarshal([]byte(`plugins: ['a']`), &c))
	require.Equal(t, []string{"a"}, c.Plugins)

	c = Default()
	require.NoError(t, Unmarshal([]byte(`gems: ['a']`), &c))
	require.Equal(t, []string{"a"}, c.Plugins)
}

func TestUnmarshal(t *testing.T) {
	c := Default()
	require.NoError(t, Unmarshal([]byte(`source: x`), &c))
	require.Equal(t, "x", c.Source)
	require.Equal(t, "./_site", c.Destination)

	c = Default()
	require.NoError(t, Unmarshal([]byte(`collections: \n- x\n-y`), &c))
	// fmt.Println(c.Collections)
}

func TestConfig_IsMarkdown(t *testing.T) {
	c := Default()
	require.True(t, c.IsMarkdown("name.md"))
	require.True(t, c.IsMarkdown("name.markdown"))
	require.False(t, c.IsMarkdown("name.html"))
}
