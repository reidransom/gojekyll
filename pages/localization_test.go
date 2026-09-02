package pages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"github.com/reidransom/jigyll/config"
	"github.com/reidransom/jigyll/tags"
	"github.com/reidransom/liquid"
	"github.com/stretchr/testify/require"
)

type pageLocalizationFake struct {
	metadata PageLocalization
}

func (f *pageLocalizationFake) PageLocalization(Page) (PageLocalization, bool) {
	return f.metadata, true
}

type localizedSiteFake struct {
	siteFake
	localization LocalizationContext
}

func (s localizedSiteFake) LocalizationContext() LocalizationContext { return s.localization }

func TestLocalizedPageDrop(t *testing.T) {
	weight := 2
	metadata := &pageLocalizationFake{}
	site := localizedSiteFake{
		siteFake:     siteFake{t: t, cfg: config.Default()},
		localization: metadata,
	}
	primary := localizedPage(t, site, "about.md", "---\npermalink: /about/\n---\nAbout")
	german := localizedPage(t, site, "ueber.md", "---\npermalink: /de/ueber/\n---\nÜber")
	metadata.metadata = PageLocalization{
		Language:       config.Locale{Key: "en", Tag: "en", Label: "English", Direction: "ltr", Default: true},
		TranslationKey: "about",
		Translations:   []Page{german},
		AllTranslations: []Page{
			primary,
			german,
		},
		Alternates: []Alternate{
			{Language: config.Locale{Key: "en", Tag: "en", Label: "English", Direction: "ltr", Default: true}, URL: "https://example.test/about/", XDefault: true},
			{Language: config.Locale{Key: "de", Tag: "de", Label: "Deutsch", Direction: "ltr", Weight: &weight}, URL: "https://example.test/de/ueber/"},
		},
	}

	engine := liquid.NewEngine()
	output, err := engine.ParseAndRender([]byte(`{{ page.language.key }}|{{ page.language.tag }}|{{ page.language.default }}|{{ page.translation_key }}|{{ page.translations.size }}|{{ page.translations[0].url }}|{{ page.all_translations.size }}|{{ page.alternates[0].tag }}|{{ page.alternates[0].url }}|{{ page.alternates[0].x_default }}`), map[string]interface{}{"page": primary})
	require.NoError(t, err)
	require.Equal(t, "en|en|true|about|1|/de/ueber/|2|en|https://example.test/about/|true", strings.TrimSpace(string(output)))
}

func TestOrdinaryPageDropOmitsLocalizationFields(t *testing.T) {
	page := localizedPage(t, siteFake{t: t, cfg: config.Default()}, "about.md", "---\n---\nAbout")
	drop := page.(liquid.Drop).ToLiquid().(tags.IterationKeyedMap)
	_, hasLanguage := drop["language"]
	_, hasTranslations := drop["translations"]
	require.False(t, hasLanguage)
	require.False(t, hasTranslations)
}

func localizedPage(t *testing.T, site Site, relativePath, contents string) Page {
	t.Helper()
	filename := filepath.Join(t.TempDir(), relativePath)
	require.NoError(t, os.WriteFile(filename, []byte(contents), 0o644))
	document, err := NewFile(site, filename, relativePath, func(bool) FrontMatter { return FrontMatter{} })
	require.NoError(t, err)
	page, ok := document.(Page)
	require.True(t, ok)
	return page
}
