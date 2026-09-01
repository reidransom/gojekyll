package server

import (
	"bytes"
	"errors"
	"fmt"
	"html"
	"io"
	"mime"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/pkg/browser"
	"github.com/reidransom/jigyll/site"
	"github.com/reidransom/liquid"
)

// Server serves the site on HTTP.
type Server struct {
	m           sync.Mutex
	Site        *site.Site
	liveReload  *liveReloadTransport
	errorOutput io.Writer
}

// Run starts a server on the address configured for the site.
func (s *Server) Run(open bool, logger func(label, value string)) error {
	cfg := s.Site.Config()
	// Only clear URL if JEKYLL_URL is not set
	if jekyllURL := os.Getenv("JEKYLL_URL"); jekyllURL != "" {
		s.Site.SetAbsoluteURL(jekyllURL)
	} else {
		s.Site.SetAbsoluteURL("")
	}
	address := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	defer listener.Close()

	logger("Server address:", "http://"+address+"/")
	if cfg.Watch {
		s.startLiveReloader()
		if err := s.watchReload(); err != nil {
			return err
		}
	}
	logger("Server running...", "press ctrl-c to stop.")
	if open {
		if err := browser.OpenURL("http://" + address); err != nil {
			fmt.Println("Error opening page:", err)
		}
	}
	return s.Serve(listener)
}

// Serve serves the site and its watch-mode LiveReload endpoints on listener.
// Closing listener stops the HTTP server and releases all LiveReload connections.
func (s *Server) Serve(listener net.Listener) error {
	if listener == nil {
		return errors.New("serve requires a listener")
	}
	if s.Site == nil {
		return errors.New("serve requires a site")
	}

	if s.Site.Config().Watch {
		s.startLiveReloader()
		defer s.stopLiveReloader()
	}

	return (&http.Server{Handler: s.routes()}).Serve(listener)
}

func (s *Server) routes() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if s.Site.Config().Watch {
			switch r.URL.Path {
			case liveReloadScriptPath:
				if err := s.currentLiveReloader().ServeScript(rw, r); err != nil {
					s.logResponseWriteError(r, err)
				}
				return
			case liveReloadWebSocketPath:
				s.currentLiveReloader().ServeWebSocket(rw, r)
				return
			}
		}
		s.handler(rw, r)
	})
}

func (s *Server) handler(rw http.ResponseWriter, r *http.Request) {
	s.m.Lock()
	defer s.m.Unlock()

	var (
		site     = s.Site
		urlpath  = r.URL.Path
		p, found = site.URLPage(urlpath)
		w        = &responseWriter{Writer: rw}
	)
	if !found {
		rw.WriteHeader(http.StatusNotFound)
		p, found = site.Routes["/404.html"]
	}
	if !found {
		if _, err := fmt.Fprintf(w, "404 page not found: %s\n", urlpath); err != nil {
			s.logResponseWriteError(r, err)
		}
		return
	}
	mimeType := mime.TypeByExtension(p.OutputExt())
	if mimeType != "" {
		rw.Header().Set("Content-Type", mimeType)
	}
	var documentWriter io.Writer = w
	if site.Config().Watch && strings.HasPrefix(mimeType, "text/html;") {
		documentWriter = NewLiveReloadInjector(documentWriter)
	}
	renderErr := site.WriteDocument(documentWriter, p)
	if renderErr == nil {
		return
	}
	if w.err != nil {
		s.logResponseWriteError(r, w.err)
		return
	}

	fmt.Fprintf(s.errorWriter(), "Error rendering %s: %s\n", urlpath, renderErr)
	eng := liquid.NewEngine()
	excerpt, path := fileErrorContext(renderErr)
	out, err := eng.ParseAndRenderString(renderErrorTemplate, liquid.Bindings{
		"error":   fmt.Sprint(renderErr),
		"excerpt": excerpt,
		"path":    path,
		"watch":   site.Config().Watch,
	})
	if err != nil {
		panic(err)
	}
	if _, err := io.WriteString(documentWriter, out); err != nil {
		s.logResponseWriteError(r, err)
	}
}

type responseWriter struct {
	io.Writer
	err error
}

func (w *responseWriter) Write(p []byte) (int, error) {
	n, err := w.Writer.Write(p)
	if err != nil && w.err == nil {
		w.err = err
	}
	return n, err
}

func (s *Server) errorWriter() io.Writer {
	if s.errorOutput != nil {
		return s.errorOutput
	}
	return os.Stderr
}

func (s *Server) logResponseWriteError(r *http.Request, err error) {
	if expectedDisconnect(err) {
		return
	}
	fmt.Fprintf(s.errorWriter(), "Error writing HTTP response for %s: %s\n", r.URL.RequestURI(), err)
}

func expectedDisconnect(err error) bool {
	return errors.Is(err, net.ErrClosed) ||
		errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, syscall.ECONNRESET)
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
