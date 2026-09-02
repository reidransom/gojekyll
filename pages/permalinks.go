package pages

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/reidransom/jigyll/utils"
)

// DefaultPermalinkPattern is the default permalink pattern for pages that aren't in a collection
const DefaultPermalinkPattern = "/:path:output_ext"

// PermalinkStyles defines built-in styles from https://jekyllrb.com/docs/permalinks/#builtinpermalinkstyles
var PermalinkStyles = map[string]string{
	"date":    "/:categories/:year/:month/:day/:title.html",
	"pretty":  "/:categories/:year/:month/:day/:title/",
	"ordinal": "/:categories/:year/:y_day/:title.html",
	"none":    "/:categories/:title.html",
}

// permalinkDateVariables maps Jekyll permalink template variable names
// to time.Format layout strings
var permalinkDateVariables = map[string]string{
	"month":      "01",
	"i_month":    "1",
	"imonth":     "1", // legacy misspelling of Jekyll's :i_month, kept for existing sites
	"day":        "02",
	"i_day":      "2",
	"hour":       "15",
	"minute":     "04",
	"second":     "05",
	"year":       "2006",
	"short_year": "06",
}

var templateVariableMatcher = regexp.MustCompile(`:\w+\b`)

// See https://jekyllrb.com/docs/permalinks/#template-variables
func (p *page) permalinkVariables() map[string]string {
	var (
		relpath = p.relPath
		root    = utils.TrimExt(relpath)
		name    = filepath.Base(root)
		// The slug front matter (set from the filename for posts) overrides the
		// filename-derived slug. Like Ruby Jekyll's UrlDrop, :slug is lowercased
		// while :title preserves case; neither uses the title front matter.
		slugSource = p.fm.String("slug", name)
		// date      = p.fileModTime
		date = p.PostDate().In(time.Local)
	)
	vars := map[string]string{
		"categories": strings.Join(p.Categories(), "/"),
		"collection": p.fm.String("collection", ""),
		"name":       utils.Slugify(name),
		"path":       "/" + root, // TODO are we removing and then adding this?
		"slug":       utils.Slugify(slugSource),
		"title":      utils.SlugifyPermalink(slugSource),
		"y_day":      fmt.Sprintf("%03d", date.YearDay()),
		// Undocumented but evident:
		"output_ext": p.OutputExt(),
	}
	for k, v := range permalinkDateVariables {
		vars[k] = date.Format(v)
	}
	// Add custom front matter variables to support custom permalinks like /:collection/:color/:path
	for k, v := range p.fm {
		if _, exists := vars[k]; !exists {
			if s, ok := v.(string); ok {
				vars[k] = utils.Slugify(s)
			}
		}
	}
	return vars
}

func isHTMLPageOutput(ext string) bool {
	switch ext {
	case ".html", ".htm", ".xhtml":
		return true
	default:
		return false
	}
}

func (p *page) defaultPagePermalinkPattern() string {
	if p.site.Config().Permalink != "pretty" ||
		!isHTMLPageOutput(p.OutputExt()) ||
		filepath.Base(utils.TrimExt(p.relPath)) == "index" {
		return DefaultPermalinkPattern
	}
	return "/:path/"
}

func (p *page) computePermalink() (string, error) {
	explicit := p.fm.String("permalink", "") != ""
	pattern := p.defaultPagePermalinkPattern()
	if explicit {
		pattern = p.fm.String("permalink", "")
		if pat, found := PermalinkStyles[pattern]; found {
			pattern = pat
		}
	}
	templateVariables := p.permalinkVariables()
	s, err := utils.SafeReplaceAllStringFunc(templateVariableMatcher, pattern, func(m string) (string, error) {
		varname := m[1:]
		value, found := templateVariables[varname]
		if !found {
			return "", fmt.Errorf("unknown variable %q in permalink template %q", varname, pattern)
		}
		return value, nil
	})
	if err != nil {
		return "", err
	}
	permalink := utils.URLPathClean("/" + s)

	// Ruby Jekyll treats an implicit HTML index page as a directory index: its
	// URL is the containing directory with a trailing slash (/, /sub/), not
	// …/index.html. The output file retains its extension (see site.WriteDoc).
	if !explicit && isHTMLPageOutput(p.OutputExt()) &&
		strings.HasSuffix(permalink, "/index"+p.OutputExt()) {
		permalink = strings.TrimSuffix(permalink, "index"+p.OutputExt()) // keep trailing slash
	}
	return permalink, nil
}

func (p *page) setPermalink() error {
	permalink, err := p.computePermalink()
	if err != nil {
		return err
	}
	if prefixer, ok := p.site.(interface{ PathPrefix() string }); ok && prefixer.PathPrefix() != "" {
		trailingSlash := strings.HasSuffix(permalink, "/")
		permalink = utils.URLPathClean("/" + strings.Trim(prefixer.PathPrefix(), "/") + "/" + strings.TrimPrefix(permalink, "/"))
		if trailingSlash && !strings.HasSuffix(permalink, "/") {
			permalink += "/"
		}
	}
	p.permalink = permalink
	return nil
}
