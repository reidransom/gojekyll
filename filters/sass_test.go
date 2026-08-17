package filters

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScssifyFilterIncludePaths(t *testing.T) {
	includeDir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(includeDir, "_palette.scss"),
		[]byte(`$accent: #123456;`),
		0o644,
	))

	css, err := scssifyFilter(
		`@import "palette"; .site-nav { color: $accent; }`,
		[]string{includeDir},
	)
	require.NoError(t, err)
	require.Contains(t, css, "color: #123456;")
}
