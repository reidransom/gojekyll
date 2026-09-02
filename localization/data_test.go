package localization

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/reidransom/jigyll/config"
	"github.com/stretchr/testify/require"
)

func TestDiscoverDataMergesValidatedFallbackOverlaysWithoutLeaks(t *testing.T) {
	dataDir := t.TempDir()
	writeDataFile(t, dataDir, "shared.yml", `
site:
  shared: common
  nested:
    shared: common
  sequence: [shared]
  explicit_null: shared
  type_change:
    shared: value
`)
	writeDataFile(t, dataDir, "locales/en/site.yml", `
nested:
  english: value
sequence: [english]
explicit_null: null
`)
	writeDataFile(t, dataDir, "locales/en/messages.yml", `
nav:
  home: Home
  english: English only
`)
	writeDataFile(t, dataDir, "locales/fr/site.yml", `
nested:
  french: value
sequence: [français]
type_change: replaced scalar
`)
	writeDataFile(t, dataDir, "locales/fr/messages.yml", `
nav:
  french: Français seulement
`)
	writeDataFile(t, dataDir, "locales/de/site.yml", `
nested:
  german: value
`)
	writeDataFile(t, dataDir, "locales/de/messages.yml", `
nav:
  home: Startseite
`)

	catalog, err := DiscoverData(dataDir, localizationConfig(t, "error"))
	require.NoError(t, err)

	shared := catalog.Shared()
	require.NotContains(t, shared, "locales")
	require.Equal(t, "shared", shared["site"].(map[string]interface{})["explicit_null"])

	de, err := catalog.Data("de")
	require.NoError(t, err)
	site := de["site"].(map[string]interface{})
	nested := site["nested"].(map[string]interface{})
	require.Equal(t, map[string]interface{}{
		"shared":  "common",
		"english": "value",
		"french":  "value",
		"german":  "value",
	}, nested)
	require.Equal(t, []interface{}{"français"}, site["sequence"])
	require.Nil(t, site["explicit_null"])
	require.Equal(t, "replaced scalar", site["type_change"])

	nested["shared"] = "mutated"
	site["sequence"].([]interface{})[0] = "mutated"
	again, err := catalog.Data("de")
	require.NoError(t, err)
	againSite := again["site"].(map[string]interface{})
	require.Equal(t, "common", againSite["nested"].(map[string]interface{})["shared"])
	require.Equal(t, []interface{}{"français"}, againSite["sequence"])

	english, err := catalog.Data("en")
	require.NoError(t, err)
	require.Equal(t, "value", english["site"].(map[string]interface{})["nested"].(map[string]interface{})["english"])
	messages, err := catalog.Messages("en")
	require.NoError(t, err)
	message, err := messages.Translate("nav.home")
	require.NoError(t, err)
	require.Equal(t, "Home", message)
	require.Equal(t, "common", catalog.Shared()["site"].(map[string]interface{})["nested"].(map[string]interface{})["shared"])
}

func TestDiscoverDataReportsAmbiguousAndReservedLocalePathsDeterministically(t *testing.T) {
	dataDir := t.TempDir()
	writeDataFile(t, dataDir, "authors.yml", "name: file\n")
	writeDataFile(t, dataDir, "authors/member.yml", "name: directory\n")
	writeDataFile(t, dataDir, "locales/unknown/messages.yml", "nav: unknown\n")
	writeDataFile(t, dataDir, "locales.yml", "not: a directory\n")

	_, err := DiscoverData(dataDir, localizationConfig(t, "error"))
	require.EqualError(t, err, "invalid localized data:\n - data key \"authors\" is defined by both \"authors\" and \"authors.yml\"\n - data key \"locales\" is defined by both \"locales\" and \"locales.yml\"\n - locale data \"locales/unknown\" names an unknown locale")
}

func TestDataCatalogRejectsUnknownLocale(t *testing.T) {
	catalog, err := DiscoverData(t.TempDir(), localizationConfig(t, "error"))
	require.NoError(t, err)

	_, err = catalog.Data("unknown")
	require.EqualError(t, err, "unknown locale \"unknown\"")
	_, err = catalog.Messages("unknown")
	require.EqualError(t, err, "unknown locale \"unknown\"")
}

func localizationConfig(t *testing.T, missingMessages string) *config.LocalizationConfig {
	t.Helper()
	c := config.Default()
	require.NoError(t, config.Unmarshal([]byte(`
localization:
  default_language: en
  missing_messages: `+missingMessages+`
  locales:
    en:
      tag: en
      label: English
    fr:
      tag: fr
      label: Français
    de:
      tag: de
      label: Deutsch
      fallbacks: [fr]
`), &c))
	return c.Localization
}

func writeDataFile(t *testing.T, dataDir, name, contents string) {
	t.Helper()
	filename := filepath.Join(dataDir, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(filename), 0o755))
	require.NoError(t, os.WriteFile(filename, []byte(contents), 0o644))
}
