package localization

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/reidransom/jigyll/config"
	"github.com/reidransom/jigyll/frontmatter"
	"github.com/stretchr/testify/require"
)

func TestDiscoverDocumentAppliesDefaultsBeforeLocaleAssignment(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "about.md")
	require.NoError(t, os.WriteFile(source, []byte("---\nlang: de\n---\nHallo\n"), 0o600))

	cfg := config.FromString(`
defaults:
  - scope:
      path: ""
      type: pages
    values:
      lang: en
      translation_key: about
`)
	document, err := DiscoverDocument(&cfg, source, "about.md", PagesNamespace, "pages")
	require.NoError(t, err)
	require.True(t, document.Included)
	require.Equal(t, "de", document.FrontMatter["lang"])
	require.Equal(t, "about", document.FrontMatter["translation_key"])

	catalog, err := BuildCatalog(registry(t), []Document{document})
	require.NoError(t, err)
	inputs := catalog.PreparedInputs()
	require.Len(t, inputs, 3)
	require.Empty(t, inputs[0].Documents)
	require.Len(t, inputs[1].Documents, 1)
	require.Equal(t, "de", inputs[1].Locale.Key)
	require.Equal(t, "about", inputs[1].Documents[0].TranslationKey)
}

func TestBuildCatalogFiltersPreparedInputsAndScopesIdentities(t *testing.T) {
	catalog, err := BuildCatalog(registry(t), []Document{
		document("posts/english.md", "posts", "en", "welcome", true),
		document("about.md", PagesNamespace, "en", "welcome", true),
		document("willkommen.md", PagesNamespace, "de", "welcome", true),
		document("legal.md", PagesNamespace, "de", "", true),
		document("hidden.md", PagesNamespace, "fr", "welcome", false),
	})
	require.NoError(t, err)

	inputs := catalog.PreparedInputs()
	require.Equal(t, []string{"fr", "de", "en"}, inputKeys(inputs))
	require.Empty(t, inputs[0].Documents)
	require.Equal(t, []string{"legal.md", "willkommen.md"}, inputSources(inputs[1]))
	require.Equal(t, []string{"about.md", "posts/english.md"}, inputSources(inputs[2]))

	pageEditions := catalog.Editions(Identity{Namespace: PagesNamespace, TranslationKey: "welcome"})
	require.Equal(t, []string{"de", "en"}, editionKeys(pageEditions))
	postEditions := catalog.Editions(Identity{Namespace: "posts", TranslationKey: "welcome"})
	require.Equal(t, []string{"en"}, editionKeys(postEditions))
}

func TestBuildCatalogReportsInvalidAssignmentsAndIncludedDuplicatesDeterministically(t *testing.T) {
	_, err := BuildCatalog(registry(t), []Document{
		document("b.md", PagesNamespace, "en", "same", true),
		document("a.md", PagesNamespace, "en", "same", true),
		document("unknown.md", PagesNamespace, "zz", "x", true),
		{Source: "key.md", Namespace: PagesNamespace, Included: true, FrontMatter: frontmatter.FrontMatter{"translation_key": 3}},
		document("excluded.md", PagesNamespace, "en", "same", false),
	})
	require.EqualError(t, err, "invalid localization catalog:\n - a.md: namespace \"pages\" translation_key \"same\" locale \"en\" has duplicate included editions: a.md and b.md\n - key.md: translation_key must be a non-empty string\n - unknown.md: lang \"zz\" does not name a configured locale")
}

func TestDiscoverDocumentExcludesUnpublishedUnlessEnabled(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "draft.md")
	require.NoError(t, os.WriteFile(source, []byte("---\npublished: false\n---\nDraft\n"), 0o600))

	cfg := config.Default()
	document, err := DiscoverDocument(&cfg, source, "draft.md", PagesNamespace, "pages")
	require.NoError(t, err)
	require.False(t, document.Included)

	cfg.Unpublished = true
	document, err = DiscoverDocument(&cfg, source, "draft.md", PagesNamespace, "pages")
	require.NoError(t, err)
	require.True(t, document.Included)
}

func registry(t *testing.T) *config.LocalizationConfig {
	t.Helper()
	deWeight, enWeight := 1, 2
	registry := &config.LocalizationConfig{
		DefaultLanguage: "en",
		Locales: map[string]config.Locale{
			"en": {Tag: "en", Label: "English", Weight: &enWeight},
			"de": {Tag: "de", Label: "Deutsch", Weight: &deWeight},
			"fr": {Tag: "fr", Label: "Français"},
		},
	}
	require.NoError(t, registry.Validate())
	return registry
}

func document(source, namespace, language, key string, included bool) Document {
	fm := frontmatter.FrontMatter{"lang": language}
	if key != "" {
		fm["translation_key"] = key
	}
	return Document{Source: source, RelativePath: source, Namespace: namespace, FrontMatter: fm, Included: included}
}

func inputKeys(inputs []PreparedInput) []string {
	keys := make([]string, len(inputs))
	for i, input := range inputs {
		keys[i] = input.Locale.Key
	}
	return keys
}

func inputSources(input PreparedInput) []string {
	sources := make([]string, len(input.Documents))
	for i, edition := range input.Documents {
		sources[i] = edition.Source
	}
	return sources
}

func editionKeys(editions []Edition) []string {
	keys := make([]string, len(editions))
	for i, edition := range editions {
		keys[i] = edition.Locale.Key
	}
	return keys
}
