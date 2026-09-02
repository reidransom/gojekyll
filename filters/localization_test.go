package filters

import (
	"fmt"
	"strings"
	"testing"

	"github.com/reidransom/liquid"
	"github.com/stretchr/testify/require"
)

type localizationContextFake struct {
	active       string
	sharedAssets map[string]bool
	translations map[string]interface{}
	routes       map[string]string
	messages     map[string]string
}

func (c localizationContextFake) ActiveLocale() string { return c.active }

func (c localizationContextFake) Translation(page interface{}, locale string) (interface{}, error) {
	if _, ok := page.(map[string]interface{}); !ok {
		return nil, fmt.Errorf("translation requires a page")
	}
	if locale != "en" && locale != "de" {
		return nil, fmt.Errorf("unknown locale %q", locale)
	}
	return c.translations[locale], nil
}

func (c localizationContextFake) LocalizedPageURL(page interface{}, locale string) (string, error) {
	if _, ok := page.(map[string]interface{}); !ok {
		return "", fmt.Errorf("localized_url requires a page or route")
	}
	if locale != "en" && locale != "de" {
		return "", fmt.Errorf("unknown locale %q", locale)
	}
	return c.routes["page:"+locale], nil
}

func (c localizationContextFake) LocalizedRouteURL(route, locale string) (string, error) {
	if locale != "en" && locale != "de" {
		return "", fmt.Errorf("unknown locale %q", locale)
	}
	localized, ok := c.routes[route+":"+locale]
	if !ok {
		return "", fmt.Errorf("unknown internal route %q", route)
	}
	return localized, nil
}

func (c localizationContextFake) IsSharedAsset(route string) bool { return c.sharedAssets[route] }

func (c localizationContextFake) Translate(key string) (string, error) {
	message, ok := c.messages[key]
	if !ok {
		return "", fmt.Errorf("missing message %q", key)
	}
	return message, nil
}

func TestLocalizedFilterRegistration(t *testing.T) {
	context := localizationContextFake{
		active:       "en",
		sharedAssets: map[string]bool{"/assets/app.css": true},
		translations: map[string]interface{}{"de": map[string]interface{}{"url": "/de/ueber/"}},
		routes: map[string]string{
			"/about/:en": "/about/",
			"/about/:de": "/de/ueber/",
			"page:en":   "/about/",
			"page:de":   "/de/ueber/",
		},
		messages: map[string]string{"nav.home": "Home"},
	}

	t.Run("resolves translation and defaults target locale", func(t *testing.T) {
		engine := localizedFilterEngine(context)
		output, err := engine.ParseAndRender([]byte(`{% assign edition = page | translation: "de" %}{{ edition.url }}|{{ page | localized_url }}|{{ "nav.home" | translate }}`), map[string]interface{}{
			"page": map[string]interface{}{"url": "/about/"},
		})
		require.NoError(t, err)
		require.Equal(t, "/de/ueber/|/about/|Home", strings.TrimSpace(string(output)))
	})

	t.Run("preserves route suffixes and exclusions", func(t *testing.T) {
		engine := localizedFilterEngine(context)
		output, err := engine.ParseAndRender([]byte(`{{ "/about/?view=full#intro" | localized_url: "de" }}|{{ "https://example.test/about/?q=1#top" | localized_url: "de" }}|{{ "//cdn.example.test/app.js" | localized_url: "de" }}|{{ "#section" | localized_url: "de" }}|{{ "/assets/app.css?v=1#main" | localized_url: "de" }}`), nil)
		require.NoError(t, err)
		require.Equal(t, "/de/ueber/?view=full#intro|https://example.test/about/?q=1#top|//cdn.example.test/app.js|#section|/assets/app.css?v=1#main", strings.TrimSpace(string(output)))
	})

	t.Run("returns nil for an omitted edition", func(t *testing.T) {
		engine := localizedFilterEngine(context)
		output, err := engine.ParseAndRender([]byte(`{% assign edition = page | translation: "en" %}{% if edition %}present{% else %}missing{% endif %}`), map[string]interface{}{
			"page": map[string]interface{}{"url": "/about/"},
		})
		require.NoError(t, err)
		require.Equal(t, "missing", strings.TrimSpace(string(output)))
	})

	t.Run("propagates invalid page locale route and message errors", func(t *testing.T) {
		for _, template := range []string{
			`{{ page | translation: "fr" }}`,
			`{{ "not-a-page" | translation: "de" }}`,
			`{{ "/missing/" | localized_url: "de" }}`,
			`{{ "missing.key" | translate }}`,
		} {
			engine := localizedFilterEngine(context)
			_, err := engine.ParseAndRender([]byte(template), map[string]interface{}{"page": map[string]interface{}{}})
			require.Errorf(t, err, template)
		}
	})
}

func TestLocalizedURLRejectsMultipleLocaleArguments(t *testing.T) {
	engine := localizedFilterEngine(localizationContextFake{active: "en"})
	_, err := engine.ParseAndRender([]byte(`{{ "/about/" | localized_url: "en", "de" }}`), nil)
	require.Error(t, err)
}

func localizedFilterEngine(context LocalizationContext) *liquid.Engine {
	engine := liquid.NewEngine()
	AddLocalizationFilters(engine, context)
	return engine
}
