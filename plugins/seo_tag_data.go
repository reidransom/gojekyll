package plugins

import (
	"fmt"
	"strings"
	"time"

	"github.com/reidransom/jigyll/config"
	"github.com/reidransom/jigyll/utils"
	"github.com/reidransom/liquid"
	"github.com/reidransom/liquid/tags"
)

var seoSiteFields = []string{"url", "twitter", "facebook", "logo", "social", "google_site_verification", "lang"}
var seoPageOrSiteFields = []string{"author", "image", "lang"}

func buildSEOTagData(page, site tags.IterationKeyedMap, cfg *config.Config) map[string]interface{} {
	pageTitle := page["title"]
	siteTitle := site["title"]
	if siteTitle == nil {
		siteTitle = site["name"]
	}

	seoTag := map[string]interface{}{
		"title?":        true,
		"title":         siteTitle,
		"canonical_url": canonicalURL(page, cfg),
		"page_locale":   pageLocale(page, site),
		"page_title":    pageTitle,
		"site_title":    siteTitle,
	}
	copySEOFields(seoTag, site, append(seoSiteFields, seoPageOrSiteFields...))
	copySEOFields(seoTag, page, seoPageOrSiteFields)
	if description := descriptionFor(page, site); description != nil {
		seoTag["description"] = description
	}
	if pageTitle != nil && siteTitle != nil && pageTitle != siteTitle {
		seoTag["title"] = fmt.Sprintf("%s | %s", pageTitle, siteTitle)
	}
	if author, ok := seoTag["author"].(string); ok {
		if data, err := utils.FollowDots(site, []string{"data", "authors", author}); err == nil && data != nil {
			seoTag["author"] = data
		}
	}

	datePublished := page["date"]
	if datePublished != nil {
		seoTag["date_published"] = datePublished
	}
	if dateModified := dateModifiedFor(page, datePublished); dateModified != nil {
		seoTag["date_modified"] = dateModified
	}
	seoType, website := seoTypeFor(page, datePublished)
	seoTag["type"] = seoType
	if website && siteTitle != nil {
		seoTag["name"] = siteTitle
	}
	seoTag["json_ld"] = makeJSONLD(seoTag)
	return seoTag
}

func copySEOFields(to, from map[string]interface{}, fields []string) {
	for _, name := range fields {
		if value := from[name]; value != nil {
			to[name] = value
		}
	}
}

func descriptionFor(page, site tags.IterationKeyedMap) interface{} {
	if description := page["description"]; description != nil {
		return description
	}
	if excerpt, ok := page["excerpt"].(string); ok && !strings.Contains(excerpt, "<") {
		return excerpt
	}
	return site["description"]
}

func pageLocale(page, site tags.IterationKeyedMap) string {
	for _, value := range []interface{}{page["lang"], site["lang"]} {
		if locale, ok := value.(string); ok && locale != "" {
			return locale
		}
	}
	return "en_US"
}

func canonicalURL(page tags.IterationKeyedMap, cfg *config.Config) string {
	if override, ok := page["canonical_url"].(string); ok && override != "" {
		return override
	}
	pageURL, _ := page["url"].(string)
	url := utils.URLJoin(cfg.AbsoluteURL, cfg.BaseURL, pageURL)
	if strings.HasSuffix(url, "/index.html") {
		return strings.TrimSuffix(url, "index.html")
	}
	return url
}

func dateModifiedFor(page tags.IterationKeyedMap, datePublished interface{}) interface{} {
	if value := pageSEOField(page, "date_modified"); value != nil {
		return value
	}
	if value := page["last_modified_at"]; value != nil {
		return value
	}
	return datePublished
}

func pageSEOField(page tags.IterationKeyedMap, field string) interface{} {
	seo := page["seo"]
	if seo == nil {
		return nil
	}
	value, err := utils.FollowDots(map[string]interface{}{"seo": seo}, []string{"seo", field})
	if err != nil {
		return nil
	}
	return value
}

func seoTypeFor(page tags.IterationKeyedMap, datePublished interface{}) (string, bool) {
	if seoType, ok := pageSEOField(page, "type").(string); ok && seoType != "" {
		return seoType, false
	}
	route, _ := page["url"].(string)
	switch route {
	case "/", "/index.htm", "/index.html", "/about/", "/about/index.htm", "/about/index.html":
		return "WebSite", true
	}
	if datePublished != nil {
		return "BlogPosting", false
	}
	return "WebPage", false
}

func makeJSONLD(seoTag map[string]interface{}) interface{} {
	jsonLD := map[string]interface{}{
		"@context": "https://schema.org",
		"@type":    seoTag["type"],
	}
	copyJSONLDField(jsonLD, "headline", seoTag["page_title"])
	copyJSONLDField(jsonLD, "description", seoTag["description"])
	copyJSONLDField(jsonLD, "url", seoTag["canonical_url"])
	copyJSONLDField(jsonLD, "name", seoTag["name"])
	if published := seoTag["date_published"]; published != nil {
		copyJSONLDField(jsonLD, "datePublished", jsonDate(published))
		copyJSONLDField(jsonLD, "dateModified", jsonDate(seoTag["date_modified"]))
	}
	if author := seoTag["author"]; author != nil {
		if m, ok := liquid.FromDrop(author).(map[string]interface{}); ok {
			author = m["name"]
		}
		copyJSONLDField(jsonLD, "author", map[string]interface{}{
			"@type": "Person",
			"name":  author,
		})
	}
	return jsonLD
}

func copyJSONLDField(to map[string]interface{}, name string, value interface{}) {
	if value != nil {
		to[name] = value
	}
}

func jsonDate(value interface{}) interface{} {
	switch value := value.(type) {
	case time.Time:
		return value.Format("2006-01-02T15:04:05-07:00")
	default:
		return value
	}
}
