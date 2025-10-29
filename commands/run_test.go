package commands

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseAndRun(t *testing.T) {
	// Clean up _site directory before test
	os.RemoveAll("testdata/site/_site")
	err := ParseAndRun([]string{"build", "-s", "testdata/site", "-q"})
	require.NoError(t, err)
}
