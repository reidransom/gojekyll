package frontmatter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFileHasFrontMatter(t *testing.T) {
	fm := func(filename string) bool {
		fm, err := FileHasFrontMatter(filename)
		require.NoError(t, err)
		return fm
	}
	require.True(t, fm("testdata/empty_fm.md"))
	require.True(t, fm("testdata/some_fm.md"))
	require.False(t, fm("testdata/no_fm.md"))
	require.True(t, fm("testdata/no_trailing_newline.md"), "should detect frontmatter even without trailing newline")
}

func TestFrontMatter_SortedStringArray(t *testing.T) {
	sorted := func(v interface{}) []string {
		fm := FrontMatter{"categories": v}
		return fm.SortedStringArray("categories")
	}
	require.Equal(t, []string{"a", "b"}, sorted("b a"))
	require.Equal(t, []string{"a", "b"}, sorted([]interface{}{"b", "a"}))
	require.Equal(t, []string{"a", "b"}, sorted([]string{"b", "a"}))
	require.Len(t, sorted(3), 0)
}

func TestRead_NoTrailingNewline(t *testing.T) {
	// Test parsing frontmatter when file doesn't end with a newline
	source := []byte("---\nlayout: archive\n---")
	firstLine := 0
	fm, err := Read(&source, &firstLine)
	require.NoError(t, err, "should parse frontmatter without trailing newline")
	require.Equal(t, "archive", fm["layout"], "should extract layout value")
	require.Equal(t, 3, firstLine, "content should start at line 3")
	require.Empty(t, source, "content should be empty after frontmatter")
}
