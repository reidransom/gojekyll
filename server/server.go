package server

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"mime"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/jaschaephraim/lrserver"
	"github.com/pkg/browser"
	"github.com/reidransom/jigyll/site"
	"github.com/reidransom/liquid"
)

// Server serves the site on HTTP.
type Server struct {
	m    sync.Mutex
	Site *site.Site
	lr   *lrserver.Server
}

// Run runs the server.
func (s *Server) Run(open bool, logger func(label, value string)) error {
	cfg := s.Site.Config()
	// Only clear URL if JEKYLL_URL is not set
	if jekyllURL := os.Getenv("JEKYLL_URL"); jekyllURL != "" {
		s.Site.SetAbsoluteURL(jekyllURL)
	} else {
		s.Site.SetAbsoluteURL("")
	}
	address := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	logger("Server address:", "http://"+address+"/")
	if cfg.Watch {
		if err := s.startLiveReloader(); err != nil {
			return err
		}
		if err := s.watchReload(); err != nil {
			return err
		}
	}
	http.HandleFunc("/", s.handler)
	c := make(chan error)
	go func() {
		c <- http.ListenAndServe(address, nil)
	}()
	logger("Server running...", "press ctrl-c to stop.")
	if open {
		if err := browser.OpenURL("http://" + address); err != nil {
			fmt.Println("Error opening page:", err)
		}
	}
	return <-c
}

func (s *Server) handler(rw http.ResponseWriter, r *http.Request) {
	s.m.Lock()
	defer s.m.Unlock()

	var (
		site     = s.Site
		urlpath  = r.URL.Path
		p, found = site.URLPage(urlpath)
	)
	if !found {
		rw.WriteHeader(http.StatusNotFound)
		p, found = site.Routes["/404.html"]
	}
	if !found {
		_, err := fmt.Fprintf(rw, "404 page not found: %s\n", urlpath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing HTTP response: %s", err)
		}
		return
	}
	mimeType := mime.TypeByExtension(p.OutputExt())
	if mimeType != "" {
		rw.Header().Set("Content-Type", mimeType)
	}
	var w io.Writer = rw
	if strings.HasPrefix(mimeType, "text/html;") {
		w = NewLiveReloadInjector(w)
	}
	err := site.WriteDocument(w, p)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error rendering %s: %s\n", urlpath, err)
		eng := liquid.NewEngine()
		excerpt, path := fileErrorContext(err)
		out, e := eng.ParseAndRenderString(renderErrorTemplate, liquid.Bindings{
			"error":   fmt.Sprint(err),
			"excerpt": excerpt,
			"path":    path,
			"watch":   site.Config().Watch,
		})
		if e != nil {
			panic(e)
		}
		if _, err := io.WriteString(w, out); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing HTTP response: %s", err)
		}
	}
}

func fileErrorContext(e error) (s, path string) {
	cause, ok := e.(liquid.SourceError)
	if !ok {
		return
	}
	path, n := cause.Path(), cause.LineNumber()
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	l0, l1 := n-4, n+4
	w := new(bytes.Buffer)
	for i := l0; i < l1; i++ {
		if i < 0 || len(lines) <= i {
			continue
		}
		var class string
		if i+1 == n {
			class = "error"
		}
		fmt.Fprintf(w, `<span class="line %s"><span class="gutter"></span><span class="lineno">%4d</span>%s<br /></span>`, class, i+1, html.EscapeString(lines[i]))
	}
	return w.String(), path
}

// renderErrorTemplate keeps browser-side build failures readable without competing
// with the error itself.
const renderErrorTemplate = `<!doctype html>
<html lang="en">
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>Render failed</title>
	<style>
		:root { color-scheme: light; line-height: 1.5; }
		body { max-width: 72rem; margin: 0 auto; padding: 2rem; color: #111; background: #fff; font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; }
		h1 { margin: 0 0 1rem; font-size: 1rem; font-weight: 600; }
		div { margin-bottom: 1rem; }
		code { font-family: ui-monospace, "SFMono-Regular", Menlo, Consolas, monospace; font-size: 0.875rem; }
		.line { display: block; }
		.line.error { color: #b42318; }
		.lineno { display: inline-block; width: 3em; margin-right: 1em; color: #767676; text-align: right; }
		footer { margin-top: 2rem; color: #767676; font-size: 0.875rem; }
	</style>
</head>
<body>
	<h1>Failed to render.</h1>
	<div>{{ error }}:</div>
	<code>{{ excerpt }}</code>
	{% if watch and path != "" %}
	<footer>Edit and save “{{ path }}” to reload this page.</footer>
	{% endif %}
</body>
</html>`
