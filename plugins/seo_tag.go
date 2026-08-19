package plugins

import (
	"bytes"
	"text/template"

	"github.com/reidransom/jigyll/config"
	"github.com/reidransom/liquid"
	"github.com/reidransom/liquid/render"
	"github.com/reidransom/liquid/tags"
	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/html"
)

type jekyllSEOTagPlugin struct {
	plugin
	site Site
	tpl  *liquid.Template
}

func init() {
	register("jekyll-seo-tag", &jekyllSEOTagPlugin{})
}

func (p *jekyllSEOTagPlugin) AfterInitSite(s Site) error {
	p.site = s
	return nil
}

func (p *jekyllSEOTagPlugin) ConfigureTemplateEngine(e *liquid.Engine) error {
	e.RegisterTag("seo", p.seoTag)
	tpl, err := e.ParseTemplate([]byte(seoTagTemplateSource))
	if err != nil {
		panic(err)
	}
	p.tpl = tpl
	return nil
}

func (p *jekyllSEOTagPlugin) seoTag(ctx render.Context) (string, error) {
	buf := new(bytes.Buffer)
	e := seoTagTemplate.Execute(buf, seoTag{p.tpl, ctx, p.site.Config()})
	return buf.String(), e
}

type seoTag struct {
	tpl *liquid.Template
	ctx render.Context
	cfg *config.Config
}

func (p seoTag) TagBody() (string, error) {
	site := liquid.FromDrop(p.ctx.Get("site")).(tags.IterationKeyedMap)
	page := liquid.FromDrop(p.ctx.Get("page")).(tags.IterationKeyedMap)
	seoTag := buildSEOTagData(page, site, p.cfg)
	bindings := map[string]interface{}{
		"page":      page,
		"site":      site,
		"jekyll":    p.ctx.Get("jekyll"),
		"paginator": p.ctx.Get("paginator"),
		"seo_tag":   seoTag,
	}
	b, err := p.tpl.Render(bindings)
	if err != nil {
		return "", err
	}
	m := minify.New()
	m.AddFunc("text/html", html.Minify)
	min := bytes.NewBuffer(make([]byte, 0, len(b)))
	if err := m.Minify("text/html", min, bytes.NewBuffer(b)); err != nil {
		return "", err
	}
	return min.String(), nil
}

// This is a separate template so it isn't minimized away.
var seoTagTemplate = template.Must(template.New("SEO tag").Parse(
	`<!-- Begin Jekyll SEO tag -->
{{.TagBody}}
<!-- End Jekyll SEO tag -->`))
