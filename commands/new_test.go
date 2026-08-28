package commands

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScaffoldNewSiteDefault(t *testing.T) {
	target := filepath.Join(t.TempDir(), "nested", "plain-site")

	require.NoError(t, scaffoldNewSite(target, ""))

	for _, path := range []string{
		"_config.yml",
		"_layouts/default.html",
		"404.html",
		"index.md",
		".gitignore",
	} {
		require.FileExists(t, filepath.Join(target, path))
	}
	for _, path := range []string{"_data", "_drafts", "_includes", "_layouts", "_posts", "_sass", "assets"} {
		info, err := os.Stat(filepath.Join(target, path))
		require.NoError(t, err)
		require.True(t, info.IsDir())
	}

	config, err := os.ReadFile(filepath.Join(target, "_config.yml"))
	require.NoError(t, err)
	require.NotContains(t, string(config), "theme:")

	require.NoError(t, ParseAndRun([]string{"build", "-s", target, "-q"}))
	home, err := os.ReadFile(filepath.Join(target, "_site", "index.html"))
	require.NoError(t, err)
	require.Contains(t, string(home), "Welcome to Jigyll")
	notFound, err := os.ReadFile(filepath.Join(target, "_site", "404.html"))
	require.NoError(t, err)
	require.Contains(t, string(notFound), "<h1>404</h1>")
}

func TestParseAndRunNew(t *testing.T) {
	target := filepath.Join(t.TempDir(), "new-site")

	require.NoError(t, ParseAndRun([]string{"new", target}))
	require.FileExists(t, filepath.Join(target, "_config.yml"))
}

func TestScaffoldNewSiteRefusesNonEmptyTarget(t *testing.T) {
	target := filepath.Join(t.TempDir(), "site")
	require.NoError(t, os.Mkdir(target, 0o755))
	original := []byte("do not change\n")
	require.NoError(t, os.WriteFile(filepath.Join(target, "keep.txt"), original, 0o644))

	err := scaffoldNewSite(target, "")
	require.Error(t, err)

	actual, readErr := os.ReadFile(filepath.Join(target, "keep.txt"))
	require.NoError(t, readErr)
	require.Equal(t, original, actual)
	assertNoStagingDirectory(t, filepath.Dir(target), filepath.Base(target))
}

func TestScaffoldNewSiteCleansUpFailedTheme(t *testing.T) {
	t.Run("invalid URL", func(t *testing.T) {
		target := filepath.Join(t.TempDir(), "invalid-theme")

		err := scaffoldNewSite(target, "https://github.com/")
		require.Error(t, err)
		require.ErrorIs(t, statError(target), os.ErrNotExist)
		assertNoStagingDirectory(t, filepath.Dir(target), filepath.Base(target))
	})

	t.Run("theme lacks default layout", func(t *testing.T) {
		repository := newThemeRepository(t, "")
		target := filepath.Join(t.TempDir(), "missing-layout")

		err := scaffoldNewSite(target, repository)
		require.Error(t, err)
		require.ErrorIs(t, statError(target), os.ErrNotExist)
		assertNoStagingDirectory(t, filepath.Dir(target), filepath.Base(target))
	})
}

func TestScaffoldNewSiteTheme(t *testing.T) {
	repository := newThemeRepository(t, "<!doctype html><main>Theme layout {{ content }}</main>\n")
	target := filepath.Join(t.TempDir(), "themed-site")

	require.NoError(t, scaffoldNewSite(target, repository))

	config, err := os.ReadFile(filepath.Join(target, "_config.yml"))
	require.NoError(t, err)
	require.Contains(t, string(config), "theme: starter-theme\n")
	_, err = os.Stat(filepath.Join(target, "_layouts", "default.html"))
	require.True(t, errors.Is(err, os.ErrNotExist))
	require.FileExists(t, filepath.Join(target, "_theme", "starter-theme", "_layouts", "default.html"))
	require.FileExists(t, filepath.Join(target, "404.html"))

	require.NoError(t, ParseAndRun([]string{"build", "-s", target, "-q"}))
	home, err := os.ReadFile(filepath.Join(target, "_site", "index.html"))
	require.NoError(t, err)
	require.Contains(t, string(home), "Theme layout")
	notFound, err := os.ReadFile(filepath.Join(target, "_site", "404.html"))
	require.NoError(t, err)
	require.Contains(t, string(notFound), "Theme layout")
}

func newThemeRepository(t *testing.T, layout string) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for theme scaffolding tests")
	}

	repository := filepath.Join(t.TempDir(), "starter-theme.git")
	runGit(t, "init", repository)
	runGit(t, "-C", repository, "config", "user.email", "test@example.com")
	runGit(t, "-C", repository, "config", "user.name", "Jigyll Test")
	if layout != "" {
		require.NoError(t, os.MkdirAll(filepath.Join(repository, "_layouts"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(repository, "_layouts", "default.html"), []byte(layout), 0o644))
	} else {
		require.NoError(t, os.WriteFile(filepath.Join(repository, "README.md"), []byte("Theme without layouts\n"), 0o644))
	}
	runGit(t, "-C", repository, "add", ".")
	runGit(t, "-C", repository, "commit", "-m", "initial theme")
	return repository
}

func runGit(t *testing.T, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	output, err := command.CombinedOutput()
	require.NoErrorf(t, err, "git %s: %s", strings.Join(args, " "), output)
}

func statError(path string) error {
	_, err := os.Stat(path)
	return err
}

func assertNoStagingDirectory(t *testing.T, parent, targetName string) {
	t.Helper()
	staging, err := filepath.Glob(filepath.Join(parent, "."+targetName+".jigyll-new-*"))
	require.NoError(t, err)
	require.Empty(t, staging)
}
