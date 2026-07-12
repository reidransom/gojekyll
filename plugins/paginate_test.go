package plugins

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIndexURLFromPaginatePath(t *testing.T) {
	require.Equal(t, "/", indexURLFromPaginatePath("/page:num"))
	require.Equal(t, "/", indexURLFromPaginatePath("page:num"))
	require.Equal(t, "/blog/", indexURLFromPaginatePath("/blog/page:num"))
	require.Equal(t, "/blog/", indexURLFromPaginatePath("/blog/page:num/"))
	require.Equal(t, "/a/b/", indexURLFromPaginatePath("/a/b/page:num.html"))
}

func TestPaginatorDrop(t *testing.T) {
	posts := make([]Page, 5)
	pagePath := func(n int) string {
		if n == 1 {
			return "/"
		}
		return map[int]string{2: "/page2/", 3: "/page3/"}[n]
	}

	p1 := paginatorDrop(1, 2, 3, posts, pagePath)
	require.Equal(t, 1, p1["page"])
	require.Equal(t, 2, p1["per_page"])
	require.Equal(t, 5, p1["total_posts"])
	require.Equal(t, 3, p1["total_pages"])
	require.Len(t, p1["posts"], 2)
	require.Nil(t, p1["previous_page"])
	require.Nil(t, p1["previous_page_path"])
	require.Equal(t, 2, p1["next_page"])
	require.Equal(t, "/page2/", p1["next_page_path"])

	p2 := paginatorDrop(2, 2, 3, posts, pagePath)
	require.Len(t, p2["posts"], 2)
	require.Equal(t, 1, p2["previous_page"])
	require.Equal(t, "/", p2["previous_page_path"])
	require.Equal(t, 3, p2["next_page"])
	require.Equal(t, "/page3/", p2["next_page_path"])

	// The last page gets the remainder and no next page.
	p3 := paginatorDrop(3, 2, 3, posts, pagePath)
	require.Len(t, p3["posts"], 1)
	require.Equal(t, 2, p3["previous_page"])
	require.Nil(t, p3["next_page"])
	require.Nil(t, p3["next_page_path"])

	// A postless site still gets a valid page 1.
	p0 := paginatorDrop(1, 2, 1, nil, pagePath)
	require.Len(t, p0["posts"], 0)
	require.Equal(t, 0, p0["total_posts"])
	require.Equal(t, 1, p0["total_pages"])
	require.Nil(t, p0["next_page"])
}

func TestPaginatePostReadSiteValidation(t *testing.T) {
	plugin := &paginatePlugin{}

	// Disabled when paginate is unset.
	s := siteFake{}
	require.NoError(t, plugin.PostReadSite(s))

	s.c.Paginate = -1
	require.Error(t, plugin.PostReadSite(s))

	s.c.Paginate = 2
	s.c.PaginatePath = "/page/"
	require.Error(t, plugin.PostReadSite(s)) // no :num

	// No index.html template page: warns and skips, but does not error.
	s.c.PaginatePath = "/page:num"
	require.NoError(t, plugin.PostReadSite(s))
}
