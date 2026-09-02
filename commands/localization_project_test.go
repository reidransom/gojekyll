package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildCommandBuildsOneLocalizedProject(t *testing.T) {
	source := t.TempDir()
	writeLocalizedBuildFile(t, source, "_config.yml", `
localization:
  default_language: en
  locales:
    en:
      tag: en
      label: English
    de:
      tag: de
      label: Deutsch
`)
	writeLocalizedBuildFile(t, source, "index.md", "---\nlang: en\n---\nEnglish home\n")
	writeLocalizedBuildFile(t, source, "willkommen.md", "---\nlang: de\npermalink: /willkommen/\n---\nDeutsch home\n")
	writeLocalizedBuildFile(t, source, "about-en.md", "---\nlang: en\npermalink: /about/\n---\nEnglish about\n")
	writeLocalizedBuildFile(t, source, "about-de.md", "---\nlang: de\npermalink: /uber-uns/\n---\nDeutsch about\n")
	writeLocalizedBuildFile(t, source, "assets/site.css", "body {}\n")

	require.NoError(t, ParseAndRun([]string{"build", "-s", source, "-q"}))

	destination := filepath.Join(source, "_site")
	require.FileExists(t, filepath.Join(destination, "index.html"))
	require.FileExists(t, filepath.Join(destination, "about", "index.html"))
	require.FileExists(t, filepath.Join(destination, "de", "willkommen", "index.html"))
	require.FileExists(t, filepath.Join(destination, "de", "uber-uns", "index.html"))
	require.FileExists(t, filepath.Join(destination, "assets", "site.css"))
	require.NoFileExists(t, filepath.Join(destination, "de", "about", "index.html"))
	require.NoFileExists(t, filepath.Join(destination, "willkommen", "index.html"))
}

func writeLocalizedBuildFile(t *testing.T, root, name, contents string) {
	t.Helper()
	path := filepath.Join(root, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o644))
}
