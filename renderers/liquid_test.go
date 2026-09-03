package renderers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/reidransom/jigyll/config"
	"github.com/reidransom/liquid"
	"github.com/stretchr/testify/require"
)

const undefinedFilterTemplate = `{{ "hello" | no_such_filter }}`

func TestLiquidFilterStrictness(t *testing.T) {
	t.Run("default and explicit false pass input through", func(t *testing.T) {
		for _, tc := range []struct {
			name   string
			config config.Config
		}{
			{name: "default", config: config.Default()},
			{name: "explicit false", config: config.FromString("liquid:\n  strict_filters: false")},
		} {
			t.Run(tc.name, func(t *testing.T) {
				out, err := renderUndefinedFilter(tc.config)
				require.NoError(t, err)
				require.Equal(t, "hello", string(out))
			})
		}
	})

	t.Run("strict filters report undefined filter", func(t *testing.T) {
		c := config.Default()
		c.Liquid.StrictFilters = true
		out, err := renderUndefinedFilter(c)
		require.Nil(t, out)
		require.EqualError(t, err, `Liquid error (line 1): undefined filter "no_such_filter" in test.html`)
	})
}

func TestLiquidRangeAcceptsArithmeticResult(t *testing.T) {
	p := Manager{cfg: config.Default()}
	tpl, err := p.makeLiquidEngine().ParseTemplateLocation(
		[]byte(`{% assign end = 6 | minus: 1 | divided_by: 5 %}{% for i in (1..end) %}{{ i }}{% endfor %}`),
		"test.html",
		1,
	)
	require.NoError(t, err)

	out, err := tpl.Render(liquid.Bindings{})
	require.NoError(t, err)
	require.Equal(t, "1", string(out))
}

func TestLiquidEngineCachesIncludedFiles(t *testing.T) {
	cfg := config.Default()
	cfg.Source = t.TempDir()
	filename := filepath.Join(cfg.Source, cfg.IncludesDir, "navigation.html")
	store := &countingTemplateStore{templates: map[string]string{
		filename: "first navigation",
	}}

	manager := Manager{cfg: cfg}
	engine := manager.makeLiquidEngine()
	engine.RegisterTemplateStore(store)
	template, err := engine.ParseTemplateLocation([]byte(`{% include navigation.html %}`), "page.html", 1)
	require.NoError(t, err)

	output, err := template.Render(nil)
	require.NoError(t, err)
	require.Equal(t, "first navigation", string(output))

	store.templates[filename] = "second navigation"
	output, err = template.Render(nil)
	require.NoError(t, err)
	require.Equal(t, "first navigation", string(output))
	require.Equal(t, 1, store.reads[filename])
}

type countingTemplateStore struct {
	templates map[string]string
	reads     map[string]int
}

func (s *countingTemplateStore) ReadTemplate(filename string) ([]byte, error) {
	if source, ok := s.templates[filename]; ok {
		if s.reads == nil {
			s.reads = make(map[string]int)
		}
		s.reads[filename]++
		return []byte(source), nil
	}
	return nil, os.ErrNotExist
}

func renderUndefinedFilter(c config.Config) ([]byte, error) {
	p := Manager{cfg: c}
	tpl, err := p.makeLiquidEngine().ParseTemplateLocation([]byte(undefinedFilterTemplate), "test.html", 1)
	if err != nil {
		return nil, err
	}
	return tpl.Render(liquid.Bindings{})
}
