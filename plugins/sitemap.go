package plugins

import (
	"bytes"
	"encoding/xml"
	"sort"
)

type sitemapPlugin struct{ plugin }

func init() {
	register("jekyll-sitemap", func() Plugin { return &sitemapPlugin{} })
}

func (p *sitemapPlugin) PostReadSite(s Site) error {
	if s.Config().Enabled() {
		return nil
	}
	s.AddHTMLPage("/sitemap.xml", sitemapTemplateSource, nil)
	if !s.HasRoute("/robots.txt") {
		s.AddHTMLPage("/robots.txt", `Sitemap: {{ "sitemap.xml" | absolute_url }}`, nil)
	}
	return nil
}

// SitemapAlternate is one published alternate for a sitemap URL entry.
type SitemapAlternate struct {
	Language string
	URL      string
	XDefault bool
}

// SitemapEntry is one aggregate sitemap record.
type SitemapEntry struct {
	URL          string
	LastModified string
	Alternates   []SitemapAlternate
}

// RenderLocalizedSitemap renders a localized aggregate sitemap. Entries and
// alternates are sorted so equivalent project generations produce identical
// artifacts regardless of document discovery order.
func RenderLocalizedSitemap(entries []SitemapEntry) string {
	entries = append([]SitemapEntry(nil), entries...)
	sort.Slice(entries, func(i, j int) bool { return entries[i].URL < entries[j].URL })

	var output bytes.Buffer
	output.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	output.WriteString(`<urlset xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"` + "\n")
	output.WriteString(`        xsi:schemaLocation="http://www.sitemaps.org/schemas/sitemap/0.9` + "\n")
	output.WriteString(`                           http://www.sitemaps.org/schemas/sitemap/0.9/sitemap.xsd"` + "\n")
	output.WriteString(`        xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"` + "\n")
	output.WriteString(`        xmlns:xhtml="http://www.w3.org/1999/xhtml">` + "\n")
	for _, entry := range entries {
		output.WriteString("  <url>\n    <loc>")
		writeSitemapEscaped(&output, entry.URL)
		output.WriteString("</loc>\n")
		if entry.LastModified != "" {
			output.WriteString("    <lastmod>")
			writeSitemapEscaped(&output, entry.LastModified)
			output.WriteString("</lastmod>\n")
		}
		alternates := append([]SitemapAlternate(nil), entry.Alternates...)
		sort.Slice(alternates, func(i, j int) bool {
			if alternates[i].Language != alternates[j].Language {
				return alternates[i].Language < alternates[j].Language
			}
			return alternates[i].URL < alternates[j].URL
		})
		for _, alternate := range alternates {
			output.WriteString(`    <xhtml:link rel="alternate" hreflang="`)
			writeSitemapEscaped(&output, alternate.Language)
			output.WriteString(`" href="`)
			writeSitemapEscaped(&output, alternate.URL)
			output.WriteString(`"/>` + "\n")
			if alternate.XDefault {
				output.WriteString(`    <xhtml:link rel="alternate" hreflang="x-default" href="`)
				writeSitemapEscaped(&output, alternate.URL)
				output.WriteString(`"/>` + "\n")
			}
		}
		output.WriteString("  </url>\n")
	}
	output.WriteString("</urlset>\n")
	return output.String()
}

func writeSitemapEscaped(output *bytes.Buffer, value string) {
	_ = xml.EscapeText(output, []byte(value))
}

// Taken verbatim from https://github.com/jekyll/jekyll-sitemap-plugin/
const sitemapTemplateSource = `<?xml version="1.0" encoding="UTF-8"?>
{% if page.xsl %}
  <?xml-stylesheet type="text/xsl" href="{{ "/sitemap.xsl" | absolute_url }}"?>
{% endif %}
<urlset xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
        xsi:schemaLocation="http://www.sitemaps.org/schemas/sitemap/0.9
                           http://www.sitemaps.org/schemas/sitemap/0.9/sitemap.xsd"
        xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  {% assign collections = site.collections | where_exp:'collection','collection.output != false' %}
  {% for collection in collections %}
    {% assign docs = collection.docs | where_exp:'doc','doc.sitemap != false' %}
    {% for doc in docs %}
    {% unless doc.robots contains 'noindex' %}
      <url>
        <loc>{{ doc.url | replace:'/index.html','/' | absolute_url | xml_escape }}</loc>
        {% if doc.last_modified_at or doc.date %}
          <lastmod>{{ doc.last_modified_at | default: doc.date | date_to_xmlschema }}</lastmod>
        {% endif %}
      </url>
    {% endunless %}
    {% endfor %}
  {% endfor %}

  {% assign pages = site.html_pages | where_exp:'doc','doc.sitemap != false' | where_exp:'doc','doc.url != "/404.html"' %}
  {% for page in pages %}
    {% unless page.robots contains 'noindex' %}
      <url>
        <loc>{{ page.url | replace:'/index.html','/' | absolute_url | xml_escape }}</loc>
        {% if page.last_modified_at %}
          <lastmod>{{ page.last_modified_at | date_to_xmlschema }}</lastmod>
        {% endif %}
      </url>
    {% endunless %}
  {% endfor %}

</urlset>`
