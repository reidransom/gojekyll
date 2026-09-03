package site

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/reidransom/jigyll/config"
	"github.com/stretchr/testify/require"
)

func TestLocalizedProjectRebuildRetainsGenerationUntilRequiredTranslationRecovers(t *testing.T) {
	source, english, french := requiredTranslationProjectFixture(t)

	base, err := FromDirectory(source, config.Flags{})
	require.NoError(t, err)
	project, _, err := BuildLocalizedProject(base)
	require.NoError(t, err)

	destination := filepath.Join(source, "_site", "guide", "index.html")
	initialOutput, err := os.ReadFile(destination)
	require.NoError(t, err)
	require.Contains(t, string(initialOutput), "first generation")
	require.FileExists(t, filepath.Join(source, "_site", "fr", "guide", "index.html"))
	requireLocalizedGeneration(t, project, "first")

	require.NoError(t, os.WriteFile(french, []byte(`---
lang: fr
translation_key: guide
published: false
permalink: /guide/
---
French guide
`), 0o644))

	replacement, _, err := project.Rebuild()
	require.ErrorContains(t, err, `namespace "pages" translation_key "guide" is missing required locale "fr"`)
	require.Nil(t, replacement)
	require.Equal(t, initialOutput, readLocalizedProjectOutput(t, destination))
	requireLocalizedGeneration(t, project, "first")

	require.NoError(t, os.WriteFile(english, []byte(`---
lang: en
translation_key: guide
generation: second
permalink: /guide/
---
second generation
`), 0o644))
	require.NoError(t, os.WriteFile(french, []byte(`---
lang: fr
translation_key: guide
permalink: /guide/
---
French guide
`), 0o644))

	replacement, _, err = project.Rebuild()
	require.NoError(t, err)
	require.NotNil(t, replacement)
	require.Contains(t, string(readLocalizedProjectOutput(t, destination)), "second generation")
	requireLocalizedGeneration(t, replacement, "second")
}

func requiredTranslationProjectFixture(t *testing.T) (source, english, french string) {
	t.Helper()
	source = t.TempDir()
	writeLocalizedProjectFile(t, source, "_config.yml", `destination: _site
localization:
  default_language: en
  required_translations: [fr]
  locales:
    en: {tag: en, label: English}
    fr: {tag: fr, label: Français}
`)
	english = filepath.Join(source, "english.md")
	writeLocalizedProjectFile(t, source, "english.md", `---
lang: en
translation_key: guide
generation: first
permalink: /guide/
---
first generation
`)
	french = filepath.Join(source, "french.md")
	writeLocalizedProjectFile(t, source, "french.md", `---
lang: fr
translation_key: guide
permalink: /guide/
---
French guide
`)
	return source, english, french
}

func requireLocalizedGeneration(t *testing.T, project *LocalizedProject, generation string) {
	t.Helper()
	_, document, found := project.URLPage("/guide/")
	require.True(t, found)
	require.Equal(t, generation, document.FrontMatter().String("generation", ""))
}

func readLocalizedProjectOutput(t *testing.T, filename string) []byte {
	t.Helper()
	contents, err := os.ReadFile(filename)
	require.NoError(t, err)
	return contents
}

func writeLocalizedProjectFile(t *testing.T, source, relative, contents string) {
	t.Helper()
	filename := filepath.Join(source, relative)
	require.NoError(t, os.MkdirAll(filepath.Dir(filename), 0o755))
	require.NoError(t, os.WriteFile(filename, []byte(contents), 0o644))
}
