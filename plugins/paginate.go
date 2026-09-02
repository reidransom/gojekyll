package plugins

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/reidransom/jigyll/utils"
)

// paginatePlugin emulates jekyll-paginate (v1): it renders the index.html
// page named by paginate_path once per page of posts, exposing a `paginator`
// template variable to each copy. See https://jekyllrb.com/docs/pagination/.
type paginatePlugin struct{ plugin }

func init() {
	register("jekyll-paginate", func() Plugin { return &paginatePlugin{} })
}

// paginatedPage is the part of the concrete page implementation that
// pagination needs beyond the Page interface.
type paginatedPage interface {
	Page
	SetTemplateVars(vars map[string]interface{})
	Clone(permalink string) Page
}

func (p *paginatePlugin) PostReadSite(s Site) error {
	cfg := s.Config()
	perPage := cfg.Paginate
	if perPage == 0 {
		return nil
	}
	if perPage < 0 {
		return fmt.Errorf("paginate must be a positive integer; got %d", perPage)
	}
	paginatePath := cfg.PaginatePath
	if !strings.Contains(paginatePath, ":num") {
		return fmt.Errorf("paginate_path %q must contain :num", paginatePath)
	}
	tmpl := findTemplatePage(s, s.LocalizedURL(indexURLFromPaginatePath(paginatePath)))
	if tmpl == nil {
		fmt.Println("Pagination: Pagination is enabled, but I couldn't find an" +
			" index.html page to use as the pagination template. Skipping pagination.")
		return nil
	}

	posts := s.Posts()
	pageCount := (len(posts) + perPage - 1) / perPage
	if pageCount < 1 {
		pageCount = 1 // a postless site still gets page 1 with an empty posts list
	}
	pagePath := func(n int) string {
		if n == 1 {
			// Page 1 is the index page itself; no page1 URL is generated.
			return tmpl.URL()
		}
		url := s.LocalizedURL(utils.URLPathClean("/" + strings.ReplaceAll(paginatePath, ":num", fmt.Sprint(n))))
		// An extensionless path is a directory index (Jekyll writes
		// page2/index.html); the trailing slash is how jigyll routes those.
		if path.Ext(url) == "" && !strings.HasSuffix(url, "/") {
			url += "/"
		}
		return url
	}
	for n := 1; n <= pageCount; n++ {
		pg := tmpl
		if n > 1 {
			url := pagePath(n)
			if s.HasRoute(url) {
				return fmt.Errorf("pagination: page %d would overwrite the existing page at %s", n, url)
			}
			pg = tmpl.Clone(url).(paginatedPage)
			s.AddPage(pg)
		}
		pg.SetTemplateVars(map[string]interface{}{
			"paginator": paginatorDrop(n, perPage, pageCount, posts, pagePath),
		})
	}
	return nil
}

// indexURLFromPaginatePath returns the URL of the index page that
// paginate_path paginates: /page:num -> /, /blog/page:num/ -> /blog/.
func indexURLFromPaginatePath(paginatePath string) string {
	prefix, _, _ := strings.Cut(paginatePath, ":num")
	dir := path.Dir("/" + strings.TrimPrefix(prefix, "/"))
	if dir == "/" {
		return "/"
	}
	return dir + "/"
}

// findTemplatePage returns the index.html page at url. Like Jekyll, only an
// actual index.html qualifies — a markdown index does not.
func findTemplatePage(s Site, url string) paginatedPage {
	for _, pg := range s.Pages() {
		if pg.URL() != url || filepath.Base(pg.Source()) != "index.html" {
			continue
		}
		if pp, ok := pg.(paginatedPage); ok {
			return pp
		}
	}
	return nil
}

// paginatorDrop builds the paginator template variable for page n.
// The fields are jekyll-paginate's: https://jekyllrb.com/docs/pagination/#liquid-attributes-available
func paginatorDrop(n, perPage, pageCount int, posts []Page, pagePath func(int) string) map[string]interface{} {
	first := (n - 1) * perPage
	last := first + perPage
	if first > len(posts) {
		first = len(posts)
	}
	if last > len(posts) {
		last = len(posts)
	}
	m := map[string]interface{}{
		"page":               n,
		"per_page":           perPage,
		"posts":              posts[first:last],
		"total_posts":        len(posts),
		"total_pages":        pageCount,
		"previous_page":      nil,
		"previous_page_path": nil,
		"next_page":          nil,
		"next_page_path":     nil,
	}
	if n > 1 {
		m["previous_page"] = n - 1
		m["previous_page_path"] = pagePath(n - 1)
	}
	if n < pageCount {
		m["next_page"] = n + 1
		m["next_page_path"] = pagePath(n + 1)
	}
	return m
}
