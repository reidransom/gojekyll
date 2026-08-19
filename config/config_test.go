package config

import (
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func TestConfig_LiquidStrictFilters(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  string
		want bool
	}{
		{name: "omitted", want: false},
		{name: "false", src: "liquid:\n  strict_filters: false", want: false},
		{name: "true", src: "liquid:\n  strict_filters: true", want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := Default()
			require.NoError(t, Unmarshal([]byte(tc.src), &c))
			require.Equal(t, tc.want, c.Liquid.StrictFilters)
		})
	}
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

func TestConfig_RemoteTheme(t *testing.T) {
	t.Run("unmarshals and exposes Liquid variable", func(t *testing.T) {
		c := Default()
		require.NoError(t, Unmarshal([]byte("remote_theme: owner/repo@0123456789012345678901234567890123456789"), &c))
		require.Equal(t, "owner/repo@0123456789012345678901234567890123456789", c.RemoteTheme)
		require.Equal(t, c.RemoteTheme, c.Variables()["remote_theme"])
	})

	t.Run("later configuration file wins", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "first.yml"), []byte("remote_theme: first/theme@0123456789012345678901234567890123456789\n"), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "second.yml"), []byte("remote_theme: second/theme@abcdefabcdefabcdefabcdefabcdefabcdefabcd\n"), 0o600))

		c := Default()
		require.NoError(t, c.FromDirectory(dir, "first.yml,second.yml"))
		require.Equal(t, "second/theme@abcdefabcdefabcdefabcdefabcdefabcdefabcd", c.RemoteTheme)
	})
}

func TestConfig_IsMarkdown(t *testing.T) {
	c := Default()
	require.True(t, c.IsMarkdown("name.md"))
	require.True(t, c.IsMarkdown("name.markdown"))
	require.False(t, c.IsMarkdown("name.html"))
}

func defaultsFromYAML(t *testing.T, src string) *Config {
	t.Helper()
	c := Default()
	require.NoError(t, Unmarshal([]byte(src), &c))
	return &c
}

func TestConfig_GetFrontMatterDefaults(t *testing.T) {
	t.Run("empty path and type matches everything", func(t *testing.T) {
		c := defaultsFromYAML(t, `
defaults:
  - scope: {path: "", type: ""}
    values: {layout: default}
`)
		for _, tc := range []struct{ typ, rel string }{
			{"", "anything/x.md"},
			{"pages", "index.html"},
			{"posts", "_posts/x.md"},
		} {
			m := c.GetFrontMatterDefaults(tc.typ, tc.rel)
			require.Equal(t, "default", m["layout"], "typ=%q rel=%q", tc.typ, tc.rel)
		}
	})

	t.Run("non-glob path is a string prefix", func(t *testing.T) {
		c := defaultsFromYAML(t, `
defaults:
  - scope: {path: "_posts"}
    values: {layout: post}
`)
		require.Equal(t, "post", c.GetFrontMatterDefaults("posts", "_posts/2020-01-01-hello.md")["layout"])
		require.Nil(t, c.GetFrontMatterDefaults("posts", "other/x.md"))
		// Jekyll quirk (preserved): a non-glob path is a raw prefix, so "_posts"
		// also matches "_postsfoo/x.md".
		require.Equal(t, "post", c.GetFrontMatterDefaults("posts", "_postsfoo/x.md")["layout"])
	})

	t.Run("glob: exact match", func(t *testing.T) {
		c := defaultsFromYAML(t, `
defaults:
  - scope: {path: "section/*/special.md"}
    values: {layout: special}
`)
		require.Equal(t, "special", c.GetFrontMatterDefaults("pages", "section/x/special.md")["layout"])
		require.Nil(t, c.GetFrontMatterDefaults("pages", "section/x/other.md"))
		require.Nil(t, c.GetFrontMatterDefaults("pages", "section/x/special.html"))
	})

	t.Run("glob: ancestor match", func(t *testing.T) {
		c := defaultsFromYAML(t, `
defaults:
  - scope: {path: "section/*"}
    values: {layout: section}
`)
		// section/* globs the section/a directory, which prefix-matches
		// section/a/deep/f.md (Jekyll's Dir.glob + prefix-test behavior).
		require.Equal(t, "section", c.GetFrontMatterDefaults("pages", "section/a/deep/f.md")["layout"])
		require.Equal(t, "section", c.GetFrontMatterDefaults("pages", "section/a")["layout"])
		require.Nil(t, c.GetFrontMatterDefaults("pages", "other/a.md"))
	})

	t.Run("glob: recursive doublestar", func(t *testing.T) {
		c := defaultsFromYAML(t, `
defaults:
  - scope: {path: "assets/**"}
    values: {image: true}
`)
		require.Equal(t, true, c.GetFrontMatterDefaults("", "assets/img/a.png")["image"])
		require.Equal(t, true, c.GetFrontMatterDefaults("", "assets/img/sub/a.png")["image"])
		require.Equal(t, true, c.GetFrontMatterDefaults("", "assets/a.png")["image"])
		require.Nil(t, c.GetFrontMatterDefaults("", "other/a.png"))
	})

	t.Run("glob: single star does not cross slash", func(t *testing.T) {
		// A partial-segment * like *.html matches only top-level files; it
		// cannot reach a nested path because no ancestor directory matches
		// the *.html pattern.
		c := defaultsFromYAML(t, `
defaults:
  - scope: {path: "*.html"}
    values: {toplevel: true}
`)
		require.Equal(t, true, c.GetFrontMatterDefaults("pages", "top.html")["toplevel"])
		require.Nil(t, c.GetFrontMatterDefaults("pages", "dir/nested.html"))
	})

	t.Run("type filtering", func(t *testing.T) {
		c := defaultsFromYAML(t, `
defaults:
  - scope: {path: "", type: "posts"}
    values: {layout: post}
`)
		require.Equal(t, "post", c.GetFrontMatterDefaults("posts", "x.md")["layout"])
		require.Nil(t, c.GetFrontMatterDefaults("pages", "x.md"))
		// A typed scope never matches an untyped document (static file).
		require.Nil(t, c.GetFrontMatterDefaults("", "x.md"))

		// A typeless scope matches an untyped document.
		c2 := defaultsFromYAML(t, `
defaults:
  - scope: {path: ""}
    values: {a: 1}
`)
		require.Equal(t, 1, c2.GetFrontMatterDefaults("", "x.md")["a"])
	})

	t.Run("precedence: longer path wins regardless of order", func(t *testing.T) {
		// Longer path defined EARLIER beats shorter later one.
		c := defaultsFromYAML(t, `
defaults:
  - scope: {path: "blog/posts"}
    values: {layout: long}
  - scope: {path: "blog"}
    values: {layout: short}
`)
		require.Equal(t, "long", c.GetFrontMatterDefaults("pages", "blog/posts/x.md")["layout"])

		// And the reverse order still yields the longer-path value.
		cRev := defaultsFromYAML(t, `
defaults:
  - scope: {path: "blog"}
    values: {layout: short}
  - scope: {path: "blog/posts"}
    values: {layout: long}
`)
		require.Equal(t, "long", cRev.GetFrontMatterDefaults("pages", "blog/posts/x.md")["layout"])
	})

	t.Run("precedence: equal-length typed beats typeless", func(t *testing.T) {
		for _, order := range []string{`typeless then typed`, `typed then typeless`} {
			var src string
			switch order {
			case `typeless then typed`:
				src = `
defaults:
  - scope: {path: "blog", type: ""}
    values: {winner: typeless}
  - scope: {path: "blog", type: "pages"}
    values: {winner: typed}
`
			case `typed then typeless`:
				src = `
defaults:
  - scope: {path: "blog", type: "pages"}
    values: {winner: typed}
  - scope: {path: "blog", type: ""}
    values: {winner: typeless}
`
			}
			c := defaultsFromYAML(t, src)
			require.Equal(t, "typed", c.GetFrontMatterDefaults("pages", "blog/x.md")["winner"], order)
		}
	})

	t.Run("precedence: equal ties resolve to later entry", func(t *testing.T) {
		c := defaultsFromYAML(t, `
defaults:
  - scope: {path: "blog"}
    values: {x: first}
  - scope: {path: "blog"}
    values: {x: second}
`)
		require.Equal(t, "second", c.GetFrontMatterDefaults("pages", "blog/x.md")["x"])
	})

	t.Run("multiple scopes merge different keys", func(t *testing.T) {
		c := defaultsFromYAML(t, `
defaults:
  - scope: {path: ""}
    values: {a: 1}
  - scope: {path: "blog"}
    values: {b: 2}
`)
		m := c.GetFrontMatterDefaults("pages", "blog/x.md")
		require.Equal(t, 1, m["a"])
		require.Equal(t, 2, m["b"])
	})

	t.Run("does not mutate entry values", func(t *testing.T) {
		c := defaultsFromYAML(t, `
defaults:
  - scope: {path: "blog"}
    values: {x: original}
`)
		c.GetFrontMatterDefaults("pages", "blog/x.md")
		// The original Values map in the config must be untouched.
		require.Equal(t, "original", c.Defaults[0].Values["x"])
		require.Len(t, c.Defaults[0].Values, 1)
	})
}
