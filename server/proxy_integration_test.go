package server

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/reidransom/jigyll/config"
	"github.com/reidransom/jigyll/site"
	"github.com/stretchr/testify/require"
)

type proxyIntegrationDocument struct {
	url, source, body string
	write             func(io.Writer) error
}

func (d proxyIntegrationDocument) URL() string             { return d.url }
func (d proxyIntegrationDocument) Source() string          { return d.source }
func (d proxyIntegrationDocument) OutputExt() string       { return path.Ext(d.url) }
func (d proxyIntegrationDocument) Published() bool         { return true }
func (d proxyIntegrationDocument) IsStatic() bool          { return false }
func (d proxyIntegrationDocument) Reload() error           { return nil }
func (d proxyIntegrationDocument) Write(w io.Writer) error {
	if d.write != nil {
		return d.write(w)
	}
	_, err := io.WriteString(w, d.body)
	return err
}

type proxiedIntegrationServer struct {
	url    string
	server *Server
}

func TestServeSupportsSameOriginLiveReloadThroughReverseProxy(t *testing.T) {
	first := startProxiedIntegrationServer(t, config.Flags{Watch: true}, integrationRoutes("first"), nil)
	second := startProxiedIntegrationServer(t, config.Flags{Watch: true}, integrationRoutes("second"), nil)

	response, err := http.Get(first.url + "/")
	require.NoError(t, err)
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Contains(t, string(body), `src="/livereload.js"`)
	require.NotContains(t, string(body), "localhost")

	response, err = http.Get(first.url + liveReloadScriptPath)
	require.NoError(t, err)
	client, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Contains(t, response.Header.Get("Content-Type"), "application/javascript")
	require.Contains(t, string(client), "LiveReload")

	firstConnection := connectLiveReload(t, first.url)
	defer firstConnection.Close()
	secondConnection := connectLiveReload(t, second.url)
	defer secondConnection.Close()
	waitForLiveReloadConnection(t, first.server)
	waitForLiveReloadConnection(t, second.server)

	first.server.currentLiveReloader().Reload("/first")
	var firstReload liveReloadReload
	require.NoError(t, firstConnection.ReadJSON(&firstReload))
	require.Equal(t, "/first", firstReload.Path)

	secondConnection.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	var unexpectedReload liveReloadReload
	require.Error(t, secondConnection.ReadJSON(&unexpectedReload))
	require.NoError(t, secondConnection.Close())

	secondConnection = connectLiveReload(t, second.url)
	defer secondConnection.Close()
	waitForLiveReloadConnection(t, second.server)
	second.server.currentLiveReloader().Reload("/second")
	var secondReload liveReloadReload
	require.NoError(t, secondConnection.ReadJSON(&secondReload))
	require.Equal(t, "/second", secondReload.Path)
}

func TestServeDeliversBatchReloadProtocolThroughReverseProxy(t *testing.T) {
	incremental := true
	tests := []struct {
		name  string
		flags config.Flags
		paths []string
		want  []liveReloadReload
	}{
		{
			name:  "full rebuild multi-file batch",
			flags: config.Flags{Watch: true},
			paths: []string{"layouts/default.html", "_data/navigation.yml", "assets/site.css"},
			want:  []liveReloadReload{{Command: "reload", Path: "", LiveCSS: true}},
		},
		{
			name:  "multiple page-like resources",
			flags: config.Flags{Watch: true, Incremental: &incremental},
			paths: []string{"index.html", "about.html"},
			want:  []liveReloadReload{{Command: "reload", Path: "", LiveCSS: true}},
		},
		{
			name:  "targeted CSS and image resources",
			flags: config.Flags{Watch: true, Incremental: &incremental},
			paths: []string{"assets/site.css", "images/logo.png"},
			want: []liveReloadReload{
				{Command: "reload", Path: "/assets/site.css", LiveCSS: true},
				{Command: "reload", Path: "/images/logo.png", LiveCSS: true},
			},
		},
		{
			name:  "mixed page and asset resources",
			flags: config.Flags{Watch: true, Incremental: &incremental},
			paths: []string{"about.html", "assets/site.css"},
			want:  []liveReloadReload{{Command: "reload", Path: "", LiveCSS: true}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proxied := startProxiedIntegrationServer(t, tt.flags, integrationRoutes("site"), nil)
			connection := connectLiveReload(t, proxied.url)
			defer connection.Close()
			waitForLiveReloadConnection(t, proxied.server)

			proxied.server.liveReloadBatch(site.FilesEvent{Paths: tt.paths}).Deliver()
			for _, want := range tt.want {
				var got liveReloadReload
				require.NoError(t, connection.ReadJSON(&got))
				require.Equal(t, want, got)
			}
			requireNoLiveReloadMessage(t, connection)
		})
	}
}

func TestServeClassifiesProxiedCancellationAndRenderFailures(t *testing.T) {
	t.Run("cancellation is quiet", func(t *testing.T) {
		var diagnostics bytes.Buffer
		started := make(chan struct{})
		release := make(chan struct{})
		writeResult := make(chan error, 1)
		var releaseOnce sync.Once
		releaseWriter := func() { releaseOnce.Do(func() { close(release) }) }
		t.Cleanup(releaseWriter)

		document := proxyIntegrationDocument{
			url:    "/slow.html",
			source: "slow.html",
			write: func(w io.Writer) error {
				if _, err := io.WriteString(w, "response started"); err != nil {
					writeResult <- err
					return err
				}
				close(started)
				<-release
				payload := strings.Repeat("x", 64<<10)
				for {
					if _, err := io.WriteString(w, payload); err != nil {
						writeResult <- err
						return err
					}
				}
			},
		}
		proxied := startProxiedIntegrationServer(t, config.Flags{}, map[string]site.Document{"/slow": document}, &diagnostics)

		connection, err := net.Dial("tcp", strings.TrimPrefix(proxied.url, "http://"))
		require.NoError(t, err)
		defer connection.Close()
		_, err = fmt.Fprintf(connection, "GET /slow HTTP/1.1\r\nHost: example.test\r\nConnection: close\r\n\r\n")
		require.NoError(t, err)
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("backend did not begin the proxied response")
		}

		status, err := bufio.NewReader(connection).ReadString('\n')
		require.NoError(t, err)
		require.Contains(t, status, "200")
		require.NoError(t, connection.Close())
		releaseWriter()

		select {
		case err := <-writeResult:
			require.Error(t, err)
		case <-time.After(5 * time.Second):
			t.Fatal("proxy cancellation did not close the upstream response")
		}
		require.Empty(t, diagnostics.String())
	})

	t.Run("render failure remains actionable", func(t *testing.T) {
		var diagnostics bytes.Buffer
		renderFailure := errors.New("unknown liquid tag")
		proxied := startProxiedIntegrationServer(t, config.Flags{}, map[string]site.Document{
			"/broken": proxyIntegrationDocument{
				url:    "/broken.html",
				source: "broken.html",
				write:  func(io.Writer) error { return renderFailure },
			},
		}, &diagnostics)

		response, err := http.Get(proxied.url + "/broken?preview=1")
		require.NoError(t, err)
		body, err := io.ReadAll(response.Body)
		require.NoError(t, err)
		require.NoError(t, response.Body.Close())
		require.Contains(t, string(body), "<h1>Failed to render.</h1>")
		require.Contains(t, string(body), renderFailure.Error())
		require.Equal(t, "Error rendering /broken: unknown liquid tag\n", diagnostics.String())
	})
}

func integrationRoutes(name string) map[string]site.Document {
	routes := map[string]site.Document{
		"/":                proxyIntegrationDocument{url: "/index.html", source: "index.html", body: "<head></head><body>" + name + "</body>"},
		"/about/":          proxyIntegrationDocument{url: "/about/index.html", source: "about.html", body: "<head></head><body>about</body>"},
		"/assets/site.css": proxyIntegrationDocument{url: "/assets/site.css", source: "assets/site.css", body: "body {}"},
		"/images/logo.png": proxyIntegrationDocument{url: "/images/logo.png", source: "images/logo.png", body: "image"},
	}
	for i := range 32 {
		url := fmt.Sprintf("/route-%d/", i)
		routes[url] = proxyIntegrationDocument{url: url + "index.html", source: fmt.Sprintf("route-%d.html", i), body: "<head></head><body>route</body>"}
	}
	return routes
}

func startProxiedIntegrationServer(t *testing.T, flags config.Flags, routes map[string]site.Document, output io.Writer) proxiedIntegrationServer {
	t.Helper()
	backendListener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := &Server{Site: site.New(flags), errorOutput: output}
	server.Site.Routes = routes
	backendStopped := make(chan error, 1)
	go func() { backendStopped <- server.Serve(backendListener) }()

	target, err := url.Parse("http://" + backendListener.Addr().String())
	require.NoError(t, err)
	proxyListener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	proxyServer := &http.Server{Handler: httputil.NewSingleHostReverseProxy(target)}
	proxyStopped := make(chan error, 1)
	go func() { proxyStopped <- proxyServer.Serve(proxyListener) }()

	t.Cleanup(func() {
		_ = proxyServer.Close()
		<-proxyStopped
		_ = backendListener.Close()
		<-backendStopped
	})
	return proxiedIntegrationServer{url: "http://" + proxyListener.Addr().String(), server: server}
}

