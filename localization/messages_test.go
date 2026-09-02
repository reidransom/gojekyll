package localization

import (
	"testing"

	"github.com/reidransom/jigyll/config"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

func TestMessageCatalogResolvesActiveAndFallbackMessages(t *testing.T) {
	dataDir := t.TempDir()
	writeDataFile(t, dataDir, "locales/en/messages.yml", `
messages:
  nav:
    home: Home
    english: English only
    sequence: [not, text]
`)
	writeDataFile(t, dataDir, "locales/fr/messages.yml", `
messages:
  nav:
    french: Français seulement
`)
	writeDataFile(t, dataDir, "locales/de/messages.yml", `
messages:
  nav:
    home: Startseite
`)

	data, err := DiscoverData(dataDir, localizationConfig(t, "error"))
	require.NoError(t, err)
	messages, err := data.Messages("de")
	require.NoError(t, err)

	message, err := messages.Translate("nav.home")
	require.NoError(t, err)
	require.Equal(t, "Startseite", message)
	message, err = messages.Translate("nav.french")
	require.NoError(t, err)
	require.Equal(t, "Français seulement", message)
	message, err = messages.Translate("nav.english")
	require.NoError(t, err)
	require.Equal(t, "English only", message)

	_, err = messages.Translate("nav.sequence")
	require.EqualError(t, err, "message \"nav.sequence\" for locale \"en\" must resolve to a scalar string, got sequence")
	_, err = messages.Translate("nav.absent")
	require.EqualError(t, err, "missing message \"nav.absent\" for locale \"de\"")
	_, err = messages.Translate("nav.")
	require.EqualError(t, err, "message key \"nav.\" must be a non-empty dotted key")
}

func TestMessageCatalogReturnsKeyWhenConfigured(t *testing.T) {
	data, err := DiscoverData(t.TempDir(), localizationConfig(t, "key"))
	require.NoError(t, err)
	messages, err := data.Messages("de")
	require.NoError(t, err)

	message, err := messages.Translate("navigation.missing")
	require.NoError(t, err)
	require.Equal(t, "navigation.missing", message)
}

func TestDiscoverDataRejectsInvalidFallbackConfigurationBeforeMessages(t *testing.T) {
	for _, tc := range []struct {
		name string
		config string
		want string
	}{
		{
			name: "unknown fallback",
			config: `
default_language: en
locales:
  en: {tag: en, label: English}
  de: {tag: de, label: Deutsch, fallbacks: [unknown]}
`,
			want: "discovering localized data: invalid localization configuration:\n - locales.de.fallbacks[0]: unknown locale \"unknown\"",
		},
		{
			name: "cycle",
			config: `
default_language: en
locales:
  en: {tag: en, label: English}
  de: {tag: de, label: Deutsch, fallbacks: [fr]}
  fr: {tag: fr, label: Français, fallbacks: [de]}
`,
			want: "discovering localized data: invalid localization configuration:\n - fallback cycle: de -> fr -> de",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			locales := &config.LocalizationConfig{}
			require.NoError(t, yaml.Unmarshal([]byte(tc.config), locales))

			_, err := DiscoverData(t.TempDir(), locales)
			require.EqualError(t, err, tc.want)
		})
	}
}
