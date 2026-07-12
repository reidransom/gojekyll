package collection

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPostsStrategy_parseFilename(t *testing.T) {
	fm := map[string]interface{}{}
	postsStrategy{}.parseFilename("_posts/2017-06-10-Hello-白法-Wörld.md", fm)
	require.Equal(t, "Hello 白法 Wörld", fm["title"])
	// The slug keeps the raw filename portion after the date, like Ruby Jekyll
	require.Equal(t, "Hello-白法-Wörld", fm["slug"])
	require.Equal(t, time.Date(2017, 6, 10, 0, 0, 0, 0, time.Local), fm["date"])
}

func TestDraftsStrategy_parseFilename(t *testing.T) {
	fm := map[string]interface{}{}
	draftsStrategy{}.parseFilename("_drafts/2017-07-01-My-Draft.md", fm)
	require.Equal(t, "My-Draft", fm["slug"])

	fm = map[string]interface{}{}
	draftsStrategy{}.parseFilename("_drafts/Dateless-Draft.md", fm)
	require.Equal(t, "Dateless-Draft", fm["slug"])
	require.Equal(t, "Dateless Draft", fm["title"])
}
