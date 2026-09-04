package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/reidransom/jigyll/config"
	"github.com/reidransom/jigyll/site"
	"github.com/stretchr/testify/require"
)

func TestLocalizedReloadRetainsFailedSnapshotAndAtomicallyReplacesRoutes(t *testing.T) {
	source := t.TempDir()
	writeLocalizedServerFile(t, source, "_config.yml", `
watch: true
localization:
  default_language: en
  locales:
    en:
      tag: en
      label: English
    de:
      tag: de
      label: Deutsch
`)
	writeLocalizedServerFile(t, source, "index.md", "---\nlang: en\npermalink: /old/\n---\nold English route\n")
	writeLocalizedServerFile(t, source, "willkommen.md", "---\nlang: de\npermalink: /willkommen/\n---\nGerman route\n")

	base, err := site.FromDirectory(source, config.Flags{Watch: true})
	require.NoError(t, err)
	server := &Server{Site: base}
	require.NoError(t, server.ensureLocalizedProject())
	require.True(t, server.reload(site.FilesEvent{Time: time.Unix(0, 0), Paths: []string{"index.md"}}))
	assertLocalizedResponse(t, server, "/old/", http.StatusOK, "old English route")
	assertLocalizedResponse(t, server, "/de/willkommen/", http.StatusOK, "German route")

	writeLocalizedServerFile(t, source, "index.md", "---\nlang: missing\npermalink: /new/\n---\nbroken route\n")
	require.False(t, server.reload(site.FilesEvent{Time: time.Unix(1, 0), Paths: []string{"index.md"}}))
	assertLocalizedResponse(t, server, "/old/", http.StatusOK, "old English route")
	assertLocalizedResponse(t, server, "/new/", http.StatusNotFound, "404 page not found")
	assertLocalizedResponse(t, server, "/de/willkommen/", http.StatusOK, "German route")

	writeLocalizedServerFile(t, source, "index.md", "---\nlang: en\npermalink: /new/\n---\nnew English route\n")
	require.True(t, server.reload(site.FilesEvent{Time: time.Unix(2, 0), Paths: []string{"index.md"}}))
	assertLocalizedResponse(t, server, "/old/", http.StatusNotFound, "404 page not found")
	assertLocalizedResponse(t, server, "/new/", http.StatusOK, "new English route")
	assertLocalizedResponse(t, server, "/de/willkommen/", http.StatusOK, "German route")
}

func TestLocalizedServerServesAcceptedDevelopmentSnapshot(t *testing.T) {
	source := t.TempDir()
	destination := filepath.Join(source, "generated")
	writeLocalizedServerFile(t, source, "_config.yml", `
watch: true
destination: generated
localization:
  default_language: en
  locales:
    en:
      tag: en
      label: English
    de:
      tag: de
      label: Deutsch
`)
	writeLocalizedServerFile(t, source, "_layouts/default.html", "<html><head></head><body>layout:{{ content }}</body></html>")
	writeLocalizedServerFile(t, source, "index.md", "---\nlang: en\npermalink: /guide/\nlayout: default\n---\nfirst English guide\n")
	writeLocalizedServerFile(t, source, "willkommen.md", "---\nlang: de\npermalink: /willkommen/\nlayout: default\n---\nGerman guide\n")
	writeLocalizedServerFile(t, source, "assets/site.css", "body { color: red; }\n")
	writeLocalizedServerFile(t, destination, "sentinel.txt", "preserve this destination")

	base, err := site.FromDirectory(source, config.Flags{Watch: true})
	require.NoError(t, err)
	server := &Server{Site: base}
	require.NoError(t, server.ensureLocalizedProject())

	response := httptest.NewRecorder()
	server.handler(response, httptest.NewRequest(http.MethodGet, "http://example.test/guide/", nil))
	require.Equal(t, http.StatusOK, response.Code)
	require.Contains(t, response.Header().Get("Content-Type"), "text/html")
	require.Contains(t, response.Body.String(), "first English guide")
	require.Contains(t, response.Body.String(), `src="/livereload.js?port="`)
	assertLocalizedResponse(t, server, "/de/willkommen/", http.StatusOK, "German guide")
	assertLocalizedResponse(t, server, "/assets/site.css", http.StatusOK, "body { color: red; }")
	require.Equal(t, "preserve this destination", string(readLocalizedServerFile(t, filepath.Join(destination, "sentinel.txt"))))
	require.NoFileExists(t, filepath.Join(destination, "guide", "index.html"))

	writeLocalizedServerFile(t, source, "index.md", "---\nlang: en\npermalink: /guide/\nlayout: default\n---\nchanged English guide\n")
	writeLocalizedServerFile(t, source, "assets/site.css", "body { color: blue; }\n")
	assertLocalizedResponse(t, server, "/guide/", http.StatusOK, "first English guide")
	assertLocalizedResponse(t, server, "/assets/site.css", http.StatusOK, "body { color: red; }")

	require.True(t, server.reload(site.FilesEvent{Time: time.Unix(1, 0), Paths: []string{"index.md", "assets/site.css"}}))
	assertLocalizedResponse(t, server, "/guide/", http.StatusOK, "changed English guide")
	assertLocalizedResponse(t, server, "/de/willkommen/", http.StatusOK, "German guide")
	assertLocalizedResponse(t, server, "/assets/site.css", http.StatusOK, "body { color: blue; }")
	require.Equal(t, "preserve this destination", string(readLocalizedServerFile(t, filepath.Join(destination, "sentinel.txt"))))
	require.NoFileExists(t, filepath.Join(destination, "guide", "index.html"))
}

func TestLocalizedProjectRejectsIncrementalMode(t *testing.T) {
	source := t.TempDir()
	writeLocalizedServerFile(t, source, "_config.yml", `
incremental: true
localization:
  default_language: en
  locales:
    en:
      tag: en
      label: English
`)

	base, err := site.FromDirectory(source, config.Flags{})
	require.NoError(t, err)
	_, _, err = site.BuildLocalizedDevelopmentProject(base)
	require.EqualError(t, err, "localized builds do not support incremental mode")
}

func assertLocalizedResponse(t *testing.T, server *Server, path string, status int, body string) {
	t.Helper()
	response := httptest.NewRecorder()
	server.handler(response, httptest.NewRequest(http.MethodGet, "http://example.test"+path, nil))
	require.Equal(t, status, response.Code)
	require.Contains(t, response.Body.String(), body)
}

func writeLocalizedServerFile(t *testing.T, root, name, contents string) {
	t.Helper()
	path := filepath.Join(root, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o644))
}

func readLocalizedServerFile(t *testing.T, path string) []byte {
	t.Helper()
	contents, err := os.ReadFile(path)
	require.NoError(t, err)
	return contents
}
