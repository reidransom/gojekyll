package site

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/reidransom/jigyll/config"
	"github.com/reidransom/jigyll/pages"
	"github.com/reidransom/liquid/tags"
	"github.com/stretchr/testify/require"
)

func readTestSiteDrop(t *testing.T) map[string]interface{} {
	site, err := FromDirectory("testdata/site1", config.Flags{})
	require.NoError(t, err)
	require.NoError(t, site.Read())
	return site.ToLiquid().(tags.IterationKeyedMap)
}

// TODO test cases for collections, categories, tags, data

func TestSite_ToLiquid(t *testing.T) {
	drop := readTestSiteDrop(t)
	docs, ok := drop["documents"].([]Page)
	require.True(t, ok, fmt.Sprintf("documents has type %T", drop["documents"]))
	require.Len(t, docs, 3)
}

func TestSite_ToLiquid_time(t *testing.T) {
	drop := readTestSiteDrop(t)
	_, ok := drop["time"].(time.Time)
	require.True(t, ok)
	// TODO read time from config if present
}

func TestSite_ToLiquid_pages(t *testing.T) {
	drop := readTestSiteDrop(t)
	ps, ok := drop["pages"]
	require.True(t, ok, fmt.Sprintf("pages has type %T", drop["pages"]))
	require.Len(t, ps, 2)

	ps, ok = drop["html_pages"]
	require.True(t, ok, fmt.Sprintf("pages has type %T", drop["pages"]))
	require.Len(t, ps, 2)
}

func TestSitePagesRespectsUnpublishedFlag(t *testing.T) {
	enabled := true
	cases := []struct {
		name   string
		flags  config.Flags
		hidden bool
	}{
		{name: "default", flags: config.Flags{}, hidden: false},
		{name: "unpublished", flags: config.Flags{Unpublished: &enabled}, hidden: true},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			site, err := FromDirectory("testdata/unpublished-pages", tt.flags)
			require.NoError(t, err)
			require.NoError(t, site.Read())

			drop := site.ToLiquid().(tags.IterationKeyedMap)
			sitePages, ok := drop["pages"].([]Page)
			require.True(t, ok, fmt.Sprintf("pages has type %T", drop["pages"]))

			pageURLs := make([]string, 0, len(sitePages))
			for _, page := range sitePages {
				pageURLs = append(pageURLs, page.URL())
			}
			require.Equal(t, tt.hidden, containsString(pageURLs, "/hidden/"))

			_, routed := site.Routes["/hidden/"]
			require.Equal(t, tt.hidden, routed)

			rendered := renderRoute(t, site, "/")
			require.Equal(t, tt.hidden, strings.Contains(rendered, "[Hidden|/hidden/]"))
		})
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func TestSite_ToLiquid_posts(t *testing.T) {
	drop := readTestSiteDrop(t)
	posts, ok := drop["posts"].([]Page)
	require.True(t, ok, fmt.Sprintf("posts has type %T", drop["posts"]))
	require.Len(t, posts, 1)
}

// The test config scopes a default to `type: pages`; it must reach regular
// pages and nothing else.
func TestSite_FrontMatterDefaults_pagesType(t *testing.T) {
	site, err := FromDirectory("testdata/site1", config.Flags{})
	require.NoError(t, err)
	require.NoError(t, site.Read())

	nonCollection := 0
	for _, p := range site.Pages() {
		fm := p.FrontMatter()
		if fm["collection"] == nil {
			nonCollection++
			require.Equal(t, "set", fm["pagedefault"], "a `type: pages` default must apply to %s", p.URL())
		} else {
			require.Nil(t, fm["pagedefault"], "a `type: pages` default must not apply to %s", p.URL())
		}
	}
	require.NotZero(t, nonCollection)
}

func TestSite_ToLiquid_categories(t *testing.T) {
	drop := readTestSiteDrop(t)
	categories, ok := drop["categories"].(map[string][]Page)
	require.True(t, ok, fmt.Sprintf("categories has type %T", drop["categories"]))
	require.Len(t, categories, 1)
	require.Len(t, categories["cat1"], 1)
}

func TestSite_ToLiquid_tags(t *testing.T) {
	drop := readTestSiteDrop(t)
	tags, ok := drop["tags"].(map[string][]Page)
	require.True(t, ok, fmt.Sprintf("tags has type %T", drop["tags"]))
	require.Len(t, tags["tag1"], 1, "site.tags must group posts by tag, not category")
	require.NotContains(t, tags, "cat1", "site.tags must not contain categories")
	require.NotContains(t, tags, "pagetag", "site.tags must only include posts, not regular pages")
}

func TestSite_ToLiquid_related_posts(t *testing.T) {
	drop := readTestSiteDrop(t)
	posts, ok := drop["related_posts"].([]Page)
	require.True(t, ok, fmt.Sprintf("related_posts has type %T", drop["related_posts"]))
	require.Len(t, posts, 1)
}

func TestSite_ToLiquid_static_files(t *testing.T) {
	drop := readTestSiteDrop(t)
	files, ok := drop["static_files"].([]*pages.StaticFile)
	require.True(t, ok, fmt.Sprintf("static_files has type %T", drop["static_files"]))
	require.Len(t, files, 1)

	f := files[0].ToLiquid().(tags.IterationKeyedMap)
	require.IsType(t, "", f["path"])
	require.IsType(t, time.Now(), f["modified_time"])
	require.Equal(t, ".html", f["extname"])
}
