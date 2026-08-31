package server

import (
	"bytes"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"syscall"
	"testing"

	"github.com/reidransom/jigyll/config"
	"github.com/reidransom/jigyll/site"
	"github.com/stretchr/testify/require"
)

type errorHandlingTestDocument struct {
	body string
	err  error
}

func (d errorHandlingTestDocument) URL() string       { return "/index.html" }
func (d errorHandlingTestDocument) Source() string    { return "" }
func (d errorHandlingTestDocument) OutputExt() string { return ".html" }
func (d errorHandlingTestDocument) Published() bool   { return true }
func (d errorHandlingTestDocument) IsStatic() bool    { return false }
func (d errorHandlingTestDocument) Reload() error     { return nil }
func (d errorHandlingTestDocument) Write(w io.Writer) error {
	if d.err != nil {
		return d.err
	}
	_, err := io.WriteString(w, d.body)
	return err
}

type errorHandlingResponseWriter struct {
	header http.Header
	err    error
	writes int
}

func (w *errorHandlingResponseWriter) Header() http.Header { return w.header }
func (w *errorHandlingResponseWriter) WriteHeader(int)      {}
func (w *errorHandlingResponseWriter) Write(p []byte) (int, error) {
	w.writes++
	return 0, w.err
}

func TestRenderFailureWritesDeveloperErrorPage(t *testing.T) {
	var diagnostics bytes.Buffer
	server := errorHandlingTestServer(errorHandlingTestDocument{err: errors.New("unknown liquid tag")}, &diagnostics)
	response := httptest.NewRecorder()

	server.handler(response, httptest.NewRequest(http.MethodGet, "http://example.test/articles", nil))

	require.Contains(t, response.Body.String(), "<h1>Failed to render.</h1>")
	require.Contains(t, response.Body.String(), "unknown liquid tag")
	require.Equal(t, 1, strings.Count(response.Body.String(), "<h1>Failed to render.</h1>"))
	require.Equal(t, "Error rendering /articles: unknown liquid tag\n", diagnostics.String())
}

func TestExpectedResponseWriteFailuresAreQuiet(t *testing.T) {
	for _, transportErr := range []error{
		&net.OpError{Op: "write", Net: "tcp", Err: syscall.EPIPE},
		&net.OpError{Op: "write", Net: "tcp", Err: syscall.ECONNRESET},
	} {
		t.Run(transportErr.Error(), func(t *testing.T) {
			var diagnostics bytes.Buffer
			server := errorHandlingTestServer(errorHandlingTestDocument{body: "site"}, &diagnostics)
			response := &errorHandlingResponseWriter{header: make(http.Header), err: transportErr}

			server.handler(response, httptest.NewRequest(http.MethodGet, "http://example.test/articles", nil))

			require.Empty(t, diagnostics.String())
			require.Equal(t, 1, response.writes)
		})
	}
}

func TestUnexpectedResponseWriteFailureIsLoggedOnceWithURL(t *testing.T) {
	var diagnostics bytes.Buffer
	server := errorHandlingTestServer(errorHandlingTestDocument{body: "site"}, &diagnostics)
	response := &errorHandlingResponseWriter{
		header: make(http.Header),
		err:    errors.New("socket write failed"),
	}

	server.handler(response, httptest.NewRequest(http.MethodGet, "http://example.test/articles?preview=1", nil))

	require.Equal(t, 1, response.writes)
	require.Equal(t, "Error writing HTTP response for /articles?preview=1: socket write failed\n", diagnostics.String())
}

func errorHandlingTestServer(document site.Document, output io.Writer) *Server {
	testSite := site.New(config.Flags{})
	testSite.Routes = map[string]site.Document{"/articles": document}
	return &Server{Site: testSite, errorOutput: output}
}
