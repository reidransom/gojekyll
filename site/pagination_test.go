package site

import (
	"bytes"
	"testing"

	"github.com/reidransom/jigyll/config"
	"github.com/stretchr/testify/require"
)

func renderRoute(t *testing.T, s *Site, url string) string {
	t.Helper()
	d, found := s.Routes[url]
	require.True(t, found, "no route %s", url)
	buf := new(bytes.Buffer)
	require.NoError(t, d.Write(buf))
	return buf.String()
}

func TestPagination(t *testing.T) {
	s, err := FromDirectory("testdata/paginate", config.Flags{})
	require.NoError(t, err)
	require.NoError(t, s.Read())

	// 5 posts at 2 per page = 3 pages: the index plus two generated copies.
	require.Contains(t, s.Routes, "/page2/")
	require.Contains(t, s.Routes, "/page3/")
	require.NotContains(t, s.Routes, "/page4/")
	require.NotContains(t, s.Routes, "/page1/")

	p1 := renderRoute(t, s, "/")
	require.Contains(t, p1, "page 1 of 3")
	require.Contains(t, p1, "total 5")
	require.Contains(t, p1, "posts: post5;post4;") // newest first
	require.Contains(t, p1, "next 2 at /page2/")
	require.Contains(t, p1, "prev  at ;") // page 1 has no previous page

	p2 := renderRoute(t, s, "/page2/")
	require.Contains(t, p2, "page 2 of 3")
	require.Contains(t, p2, "posts: post3;post2;")
	require.Contains(t, p2, "prev 1 at /;") // page 2 points back at the index, not /page1/
	require.Contains(t, p2, "next 3 at /page3/")

	p3 := renderRoute(t, s, "/page3/")
	require.Contains(t, p3, "page 3 of 3")
	require.Contains(t, p3, "posts: post1;")
	require.Contains(t, p3, "prev 2 at /page2/")
	require.Contains(t, p3, "next  at ;") // last page has no next page

	// Generated pages appear in site.pages, like Jekyll.
	var urls []string
	for _, p := range s.nonCollectionPages {
		urls = append(urls, p.URL())
	}
	require.Contains(t, urls, "/page2/")
	require.Contains(t, urls, "/page3/")
}

func TestPaginationDisabled(t *testing.T) {
	// site1 has posts and an unset paginate; no pages are generated and
	// paginator renders empty.
	s, err := FromDirectory("testdata/site1", config.Flags{})
	require.NoError(t, err)
	require.NoError(t, s.Read())
	for url := range s.Routes {
		require.NotContains(t, url, "page2")
	}
}
