package site

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/reidransom/jigyll/config"
	"github.com/stretchr/testify/require"
)

func TestReadDataFiles(t *testing.T) {
	source := t.TempDir()
	dataDir := filepath.Join(source, "_data")
	require.NoError(t, os.MkdirAll(filepath.Join(dataDir, "orgs"), 0o755))
	// "orgs" sorts before "zebras.yml"; a subdirectory must not stop the scan.
	require.NoError(t, os.WriteFile(filepath.Join(dataDir, "authors.yml"), []byte("dave:\n  name: David Smith\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dataDir, "orgs", "jekyll.yml"), []byte("name: Jekyll\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dataDir, "zebras.yml"), []byte("- marty\n"), 0o644))

	s := New(config.Flags{})
	s.cfg.Source = source
	require.NoError(t, s.readDataFiles())

	require.Contains(t, s.data, "authors")
	require.Contains(t, s.data, "zebras", "files sorted after a subdirectory must still be read")

	orgs, ok := s.data["orgs"].(map[string]interface{})
	require.True(t, ok, "subdirectory should be namespaced as a map")
	jekyll, ok := orgs["jekyll"].(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, "Jekyll", jekyll["name"])
}

func TestReadDataFilesMissingDir(t *testing.T) {
	s := New(config.Flags{})
	s.cfg.Source = t.TempDir()
	require.NoError(t, s.readDataFiles())
	require.Empty(t, s.data)
}
