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

func TestBuildCatalogReportsDocumentDiagnosticsUsingRelativePaths(t *testing.T) {
	_, err := BuildCatalog(requiredRegistry(t, "fr"), []Document{{
		Source:       filepath.Join(t.TempDir(), "about.md"),
		RelativePath: "about.md",
		Namespace:    PagesNamespace,
		FrontMatter:  frontmatter.FrontMatter{"lang": "en"},
		Included:     true,
	}})
	require.EqualError(t, err, `invalid localization catalog:
 - about.md: default-locale document requires a translation_key or translation_exempt covering required locale "fr"`)
}

func TestBuildCatalogLeavesRequiredTranslationPolicyDisabledBehaviorUnchanged(t *testing.T) {
	defaultWithoutKey := document("default.md", PagesNamespace, "en", "", true)
	defaultWithoutKey.FrontMatter["translation_exempt"] = true
	keyedWithoutTarget := document("keyed.md", PagesNamespace, "en", "keyed", true)
	keyedWithoutTarget.FrontMatter["translation_exempt"] = []interface{}{3}

	_, err := BuildCatalog(registry(t), []Document{defaultWithoutKey, keyedWithoutTarget})
	require.NoError(t, err)
}

func TestBuildCatalogRequiresKeysOrCompleteExemptionsForDefaultEditions(t *testing.T) {
	registry := requiredRegistry(t, "de", "fr")
	keyless := document("about.md", PagesNamespace, "en", "", true)

	_, err := BuildCatalog(registry, []Document{keyless})
	require.EqualError(t, err, "invalid localization catalog:\n - about.md: default-locale document requires a translation_key or translation_exempt covering required locale \"de\"\n - about.md: default-locale document requires a translation_key or translation_exempt covering required locale \"fr\"")

	keyless.FrontMatter["translation_exempt"] = []interface{}{"fr", "de"}
	_, err = BuildCatalog(registry, []Document{keyless})
	require.NoError(t, err)

	static := document("static.md", PagesNamespace, "en", "", true)
	static.Static = true
	excluded := document("excluded.md", PagesNamespace, "en", "", false)
	_, err = BuildCatalog(registry, []Document{static, excluded})
	require.NoError(t, err)
}

func TestBuildCatalogRequiresConfiguredTargetInEachNamespace(t *testing.T) {
	registry := requiredRegistry(t, "fr")
	_, err := BuildCatalog(registry, []Document{
		document("about.en.md", PagesNamespace, "en", "shared", true),
		document("about.fr.md", PagesNamespace, "fr", "shared", true),
		document("posts/about.en.md", "posts", "en", "shared", true),
		document("de-only.md", PagesNamespace, "de", "de-only", true),
		document("fr-only.md", PagesNamespace, "fr", "fr-only", true),
	})
	require.EqualError(t, err, "invalid localization catalog:\n - namespace \"posts\" translation_key \"shared\" is missing required locale \"fr\"")

	_, err = BuildCatalog(registry, []Document{
		document("about.en.md", PagesNamespace, "en", "shared", true),
		document("about.fr.md", PagesNamespace, "fr", "shared", true),
		document("posts/about.en.md", "posts", "en", "shared", true),
		document("posts/about.fr.md", "posts", "fr", "shared", true),
		document("de-only.md", PagesNamespace, "de", "de-only", true),
		document("fr-only.md", PagesNamespace, "fr", "fr-only", true),
	})
	require.NoError(t, err)
}

func TestBuildCatalogRequiresFrenchDespiteDanishOptionalSibling(t *testing.T) {
	registry := requiredRegistry(t, "fr")
	registry.Locales["da"] = config.Locale{Tag: "da", Label: "Dansk"}
	require.NoError(t, registry.Validate())

	_, err := BuildCatalog(registry, []Document{
		document("about.en.md", PagesNamespace, "en", "about", true),
		document("about.da.md", PagesNamespace, "da", "about", true),
	})
	require.EqualError(t, err, `invalid localization catalog:
 - namespace "pages" translation_key "about" is missing required locale "fr"`)
}

func TestBuildCatalogAcceptsValidExemptionsAndRejectsRedundantOnes(t *testing.T) {
	registry := requiredRegistry(t, "de", "fr")
	defaultEdition := document("about.en.md", PagesNamespace, "en", "about", true)
	defaultEdition.FrontMatter["translation_exempt"] = []string{"fr"}

	_, err := BuildCatalog(registry, []Document{
		defaultEdition,
		document("about.de.md", PagesNamespace, "de", "about", true),
	})
	require.NoError(t, err)

	_, err = BuildCatalog(requiredRegistry(t, "fr"), []Document{
		defaultEdition,
		document("about.fr.md", PagesNamespace, "fr", "about", true),
	})
	require.EqualError(t, err, "invalid localization catalog:\n - about.en.md: translation_exempt for required locale \"fr\" is redundant because namespace \"pages\" translation_key \"about\" has an included edition")
}

func TestBuildCatalogAggregatesInvalidExemptionsAndIndependentProblems(t *testing.T) {
	registry := requiredRegistry(t, "fr")
	malformed := document("a.md", PagesNamespace, "en", "", true)
	malformed.FrontMatter["translation_exempt"] = "fr"
	invalidEntries := document("b.md", PagesNamespace, "en", "b", true)
	invalidEntries.FrontMatter["translation_exempt"] = []interface{}{3, "unknown", "de", "fr", "fr"}
	nonDefault := document("c.md", PagesNamespace, "fr", "c", true)
	nonDefault.FrontMatter["translation_exempt"] = "fr"
	redundant := document("d.md", PagesNamespace, "en", "d", true)
	redundant.FrontMatter["translation_exempt"] = []string{"fr"}
	invalidKey := document("key.md", PagesNamespace, "en", "", true)
	invalidKey.FrontMatter["translation_key"] = 3
	invalidKey.FrontMatter["translation_exempt"] = []string{"unknown"}

	_, err := BuildCatalog(registry, []Document{
		malformed,
		invalidEntries,
		nonDefault,
		redundant,
		document("d.fr.md", PagesNamespace, "fr", "d", true),
		invalidKey,
	})
	require.EqualError(t, err, `invalid localization catalog:
 - a.md: default-locale document requires a translation_key or translation_exempt covering required locale "fr"
 - a.md: translation_exempt must be a sequence of locale-key strings
 - b.md: translation_exempt[0] must be a locale-key string
 - b.md: translation_exempt[1]: unknown locale "unknown"
 - b.md: translation_exempt[2]: locale "de" is not required
 - b.md: translation_exempt[4]: duplicate locale "fr"
 - c.md: translation_exempt is only allowed on default locale "en" editions
 - c.md: translation_exempt must be a sequence of locale-key strings
 - d.md: translation_exempt for required locale "fr" is redundant because namespace "pages" translation_key "d" has an included edition
 - key.md: translation_exempt[0]: unknown locale "unknown"
 - key.md: translation_key must be a non-empty string
 - namespace "pages" translation_key "b" is missing required locale "fr"`)
}

func TestBuildCatalogSortsPolicyAndExistingProblemsDeterministically(t *testing.T) {
	registry := requiredRegistry(t, "fr")
	documents := []Document{
		document("z.md", PagesNamespace, "en", "z", true),
		document("z.fr.md", PagesNamespace, "fr", "z", true),
		document("b.md", PagesNamespace, "en", "same", true),
		document("a.md", PagesNamespace, "en", "same", true),
		document("unknown.md", PagesNamespace, "zz", "x", true),
		document("keyless.md", PagesNamespace, "en", "", true),
	}
	documents[0].FrontMatter["translation_exempt"] = []string{"fr"}

	_, first := BuildCatalog(registry, documents)
	_, second := BuildCatalog(registry, documents)
	require.EqualError(t, first, `invalid localization catalog:
 - a.md: namespace "pages" translation_key "same" locale "en" has duplicate included editions: a.md and b.md
 - keyless.md: default-locale document requires a translation_key or translation_exempt covering required locale "fr"
 - namespace "pages" translation_key "same" is missing required locale "fr"
 - unknown.md: lang "zz" does not name a configured locale
 - z.md: translation_exempt for required locale "fr" is redundant because namespace "pages" translation_key "z" has an included edition`)
	require.EqualError(t, second, first.Error())
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

func requiredRegistry(t *testing.T, locales ...string) *config.LocalizationConfig {
	t.Helper()
	registry := registry(t)
	registry.RequiredTranslations = append([]string(nil), locales...)
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
