package plugins

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/reidransom/jigyll/config"
	"github.com/reidransom/jigyll/filters"
	"github.com/reidransom/liquid"
	"github.com/reidransom/liquid/tags"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

func TestSEOTag(t *testing.T) {
	cfg := config.Default()
	cfg.AbsoluteURL = "https://example.com"
	cfg.BaseURL = "/docs"
	site := tags.IterationKeyedMap{
		"title":       "Just the Docs",
		"description": "Documentation theme",
	}

	t.Run("renders an undated documentation page", func(t *testing.T) {
		document := renderSEOTag(t, cfg, site, tags.IterationKeyedMap{
			"title": "Line Numbers",
			"url":   "/line-numbers/",
		})

		requireMeta(t, document, map[string]string{"name": "generator", "content": "Jekyll vdevelop (jigyll)"})
		requireMeta(t, document, map[string]string{"name": "description", "content": "Documentation theme"})
		requireMeta(t, document, map[string]string{"name": "twitter:description", "property": "og:description", "content": "Documentation theme"})
		requireMeta(t, document, map[string]string{"property": "og:type", "content": "website"})
		requireNoMeta(t, document, "property", "article:published_time")
		requireNoMeta(t, document, "property", "article:modified_time")
		requireMeta(t, document, map[string]string{"name": "twitter:card", "content": "summary"})
		requireMeta(t, document, map[string]string{"name": "twitter:title", "content": "Line Numbers"})
		requireURLFields(t, document, "https://example.com/docs/line-numbers/")

		jsonLD := jsonLD(t, document)
		require.Equal(t, "https://schema.org", jsonLD["@context"])
		require.Equal(t, "WebPage", jsonLD["@type"])
		require.Equal(t, "Documentation theme", jsonLD["description"])
		require.Equal(t, "Line Numbers", jsonLD["headline"])
		require.Equal(t, "https://example.com/docs/line-numbers/", jsonLD["url"])
		require.NotContains(t, jsonLD, "datePublished")
		require.NotContains(t, jsonLD, "dateModified")
	})

	t.Run("renders a home page as a website", func(t *testing.T) {
		document := renderSEOTag(t, cfg, site, tags.IterationKeyedMap{
			"title": "Just the Docs",
			"url":   "/",
		})

		requireMeta(t, document, map[string]string{"property": "og:type", "content": "website"})
		requireMeta(t, document, map[string]string{"name": "twitter:card", "content": "summary"})
		requireMeta(t, document, map[string]string{"name": "twitter:title", "content": "Just the Docs"})
		requireNoMeta(t, document, "property", "article:published_time")
		requireNoMeta(t, document, "property", "article:modified_time")

		jsonLD := jsonLD(t, document)
		require.Equal(t, "WebSite", jsonLD["@type"])
		require.Equal(t, "Just the Docs", jsonLD["name"])
	})

	t.Run("prefers page descriptions and plain excerpts", func(t *testing.T) {
		tests := []struct {
			name     string
			page     tags.IterationKeyedMap
			expected string
		}{
			{
				name:     "description",
				page:     tags.IterationKeyedMap{"title": "Guide", "url": "/guide/", "description": "Page description", "excerpt": "Page excerpt"},
				expected: "Page description",
			},
			{
				name:     "excerpt",
				page:     tags.IterationKeyedMap{"title": "Guide", "url": "/guide/", "excerpt": "Page excerpt"},
				expected: "Page excerpt",
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				document := renderSEOTag(t, cfg, site, test.page)
				requireMeta(t, document, map[string]string{"name": "description", "content": test.expected})
				requireMeta(t, document, map[string]string{"name": "twitter:description", "property": "og:description", "content": test.expected})
			})
		}
	})

	t.Run("renders explicitly dated documents as articles", func(t *testing.T) {
		date := time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC)
		document := renderSEOTag(t, cfg, site, tags.IterationKeyedMap{
			"title": "Release Notes",
			"url":   "/release-notes/",
			"date":  date,
		})

		requireMeta(t, document, map[string]string{"property": "og:type", "content": "article"})
		requireMeta(t, document, map[string]string{"property": "article:published_time", "content": "2024-01-02T03:04:05+00:00"})
		requireMeta(t, document, map[string]string{"property": "article:modified_time", "content": "2024-01-02T03:04:05+00:00"})

		jsonLD := jsonLD(t, document)
		require.Equal(t, "BlogPosting", jsonLD["@type"])
		require.Equal(t, "2024-01-02T03:04:05+00:00", jsonLD["datePublished"])
		require.Equal(t, "2024-01-02T03:04:05+00:00", jsonLD["dateModified"])
	})

	t.Run("honors SEO type and modification overrides", func(t *testing.T) {
		published := time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC)
		modified := time.Date(2024, time.February, 3, 4, 5, 0, 0, time.UTC)
		document := renderSEOTag(t, cfg, site, tags.IterationKeyedMap{
			"title": "Release Notes",
			"url":   "/release-notes/",
			"date":  published,
			"seo": tags.IterationKeyedMap{
				"type":          "Article",
				"date_modified": modified,
			},
		})

		requireMeta(t, document, map[string]string{"property": "article:modified_time", "content": "2024-02-03T04:05:00+00:00"})
		jsonLD := jsonLD(t, document)
		require.Equal(t, "Article", jsonLD["@type"])
		require.Equal(t, "2024-02-03T04:05:00+00:00", jsonLD["dateModified"])
	})

	t.Run("uses one canonical URL for every metadata representation", func(t *testing.T) {
		tests := []struct {
			name     string
			cfg      config.Config
			page     tags.IterationKeyedMap
			expected string
		}{
			{
				name:     "pretty route",
				cfg:      cfg,
				page:     tags.IterationKeyedMap{"title": "Guide", "url": "/guide/"},
				expected: "https://example.com/docs/guide/",
			},
			{
				name:     "terminal index file",
				cfg:      cfg,
				page:     tags.IterationKeyedMap{"title": "Home", "url": "/guide/index.html"},
				expected: "https://example.com/docs/guide/",
			},
			{
				name: "base URL",
				cfg: config.Config{
					AbsoluteURL: "https://example.com",
					BaseURL:     "/reference",
				},
				page:     tags.IterationKeyedMap{"title": "Guide", "url": "/guide/"},
				expected: "https://example.com/reference/guide/",
			},
			{
				name:     "canonical override",
				cfg:      cfg,
				page:     tags.IterationKeyedMap{"title": "Guide", "url": "/guide/", "canonical_url": "https://canonical.example/guide"},
				expected: "https://canonical.example/guide",
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				document := renderSEOTag(t, test.cfg, site, test.page)
				requireURLFields(t, document, test.expected)
			})
		}
	})

	t.Run("preserves conditional metadata branches", func(t *testing.T) {
		conditionalSite := tags.IterationKeyedMap{
			"title":       "Just the Docs",
			"description": "Documentation theme",
			"twitter": tags.IterationKeyedMap{
				"username": "@justthedocs",
			},
			"facebook": tags.IterationKeyedMap{
				"app_id": "1234",
			},
			"webmaster_verifications": tags.IterationKeyedMap{
				"google": "verify-me",
			},
		}
		document := renderSEOTag(t, cfg, conditionalSite, tags.IterationKeyedMap{
			"title": "Guide",
			"url":   "/guide/",
		})

		requireMeta(t, document, map[string]string{"name": "twitter:card", "content": "summary"})
		requireMeta(t, document, map[string]string{"name": "twitter:title", "content": "Guide"})
		requireMeta(t, document, map[string]string{"name": "twitter:site", "content": "@justthedocs"})
		requireMeta(t, document, map[string]string{"property": "fb:app_id", "content": "1234"})
		requireMeta(t, document, map[string]string{"name": "google-site-verification", "content": "verify-me"})
		require.Len(t, elementsWithAttributes(document, "meta", map[string]string{"name": "twitter:card"}), 1)
		require.Len(t, elementsWithAttributes(document, "meta", map[string]string{"name": "twitter:title"}), 1)
	})
}

func renderSEOTag(t *testing.T, cfg config.Config, site, page tags.IterationKeyedMap) *html.Node {
	t.Helper()

	engine := liquid.NewEngine()
	filters.AddJekyllFilters(engine, &cfg)
	names := []string{"jekyll-seo-tag"}
	installed, err := Install(names, siteFake{cfg, engine})
	require.NoError(t, err)
	require.NoError(t, installed[names[0]].ConfigureTemplateEngine(engine))
	rendered, err := engine.ParseAndRenderString(`{% seo %}`, liquid.Bindings{
		"jekyll": map[string]string{"version": "develop (jigyll)"},
		"page":   page,
		"site":   site,
	})
	require.NoError(t, err)
	document, parseErr := html.Parse(strings.NewReader(rendered))
	require.NoError(t, parseErr)
	return document
}

func requireURLFields(t *testing.T, document *html.Node, url string) {
	t.Helper()
	require.Len(t, elementsWithAttributes(document, "link", map[string]string{"rel": "canonical", "href": url}), 1)
	requireMeta(t, document, map[string]string{"property": "og:url", "content": url})
	require.Equal(t, url, jsonLD(t, document)["url"])
}

func requireMeta(t *testing.T, document *html.Node, expected map[string]string) {
	t.Helper()
	require.Len(t, elementsWithAttributes(document, "meta", expected), 1, expected)
}

func requireNoMeta(t *testing.T, document *html.Node, attribute, value string) {
	t.Helper()
	require.Empty(t, elementsWithAttributes(document, "meta", map[string]string{attribute: value}))
}

func elementsWithAttributes(document *html.Node, tag string, expected map[string]string) []*html.Node {
	var matching []*html.Node
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == tag && hasAttributes(node, expected) {
			matching = append(matching, node)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(document)
	return matching
}

func hasAttributes(node *html.Node, expected map[string]string) bool {
	for name, value := range expected {
		found := false
		for _, attribute := range node.Attr {
			if attribute.Key == name && attribute.Val == value {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func jsonLD(t *testing.T, document *html.Node) map[string]interface{} {
	t.Helper()
	scripts := elementsWithAttributes(document, "script", map[string]string{"type": "application/ld+json"})
	require.Len(t, scripts, 1)

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(textContent(scripts[0])), &payload))
	return payload
}

func textContent(node *html.Node) string {
	var text strings.Builder
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Type == html.TextNode {
			text.WriteString(node.Data)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(node)
	return text.String()
}
