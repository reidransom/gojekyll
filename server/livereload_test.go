package server

import (
	"io"
	"net"
	"net/http"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/reidransom/jigyll/config"
	"github.com/reidransom/jigyll/site"
	"github.com/stretchr/testify/require"
)

type serverTestDocument struct {
	url, body string
}

func (d serverTestDocument) URL() string             { return d.url }
func (d serverTestDocument) Source() string          { return "" }
func (d serverTestDocument) OutputExt() string       { return path.Ext(d.url) }
func (d serverTestDocument) Published() bool         { return true }
func (d serverTestDocument) IsStatic() bool          { return false }
func (d serverTestDocument) Reload() error           { return nil }
func (d serverTestDocument) Write(w io.Writer) error { _, err := io.WriteString(w, d.body); return err }

func TestWatchServerOwnsLiveReloadRoutes(t *testing.T) {
	serverURL, server := startTestServer(t, true)

	response, err := http.Get(serverURL + "/")
	require.NoError(t, err)
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	require.Contains(t, string(body), `src="/livereload.js"`)

	response, err = http.Get(serverURL + liveReloadScriptPath)
	require.NoError(t, err)
	client, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Contains(t, string(client), "LiveReload")
	require.NotContains(t, string(client), "site-controlled script")

	connection := connectLiveReload(t, serverURL)
	waitForLiveReloadConnection(t, server)
	server.currentLiveReloader().Reload("/")
	var reload liveReloadReload
	require.NoError(t, connection.ReadJSON(&reload))
	require.Equal(t, liveReloadReload{Command: "reload", Path: "/", LiveCSS: true}, reload)
	require.NoError(t, connection.Close())
}

func TestNonWatchServerDoesNotOwnLiveReloadRoutes(t *testing.T) {
	serverURL, _ := startTestServer(t, false)

	response, err := http.Get(serverURL + "/")
	require.NoError(t, err)
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	require.NotContains(t, string(body), "livereload.js")

	response, err = http.Get(serverURL + liveReloadScriptPath)
	require.NoError(t, err)
	body, err = io.ReadAll(response.Body)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "site-controlled script", string(body))
}

func TestLiveReloadConnectionsAreIsolatedPerServer(t *testing.T) {
	firstURL, first := startTestServer(t, true)
	secondURL, second := startTestServer(t, true)
	firstConnection := connectLiveReload(t, firstURL)
	secondConnection := connectLiveReload(t, secondURL)
	defer firstConnection.Close()
	defer secondConnection.Close()
	waitForLiveReloadConnection(t, first)
	waitForLiveReloadConnection(t, second)

	first.currentLiveReloader().Reload("/first")
	var firstReload liveReloadReload
	require.NoError(t, firstConnection.ReadJSON(&firstReload))
	require.Equal(t, "/first", firstReload.Path)

	secondConnection.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	var secondReload liveReloadReload
	require.Error(t, secondConnection.ReadJSON(&secondReload))
	require.NoError(t, secondConnection.Close())
	secondConnection = connectLiveReload(t, secondURL)
	defer secondConnection.Close()
	waitForLiveReloadConnection(t, second)

	second.currentLiveReloader().Reload("/second")
	require.NoError(t, secondConnection.ReadJSON(&secondReload))
	require.Equal(t, "/second", secondReload.Path)
}

func startTestServer(t *testing.T, watch bool) (string, *Server) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	s := site.New(config.Flags{Watch: watch})
	s.Routes = map[string]site.Document{
		"/": serverTestDocument{url: "/index.html", body: "<head></head><body>site</body>"},
		liveReloadScriptPath: serverTestDocument{
			url:  liveReloadScriptPath,
			body: "site-controlled script",
		},
	}
	server := &Server{Site: s}
	stopped := make(chan error, 1)
	go func() { stopped <- server.Serve(listener) }()
	t.Cleanup(func() {
		require.NoError(t, listener.Close())
		<-stopped
	})
	return "http://" + listener.Addr().String(), server
}

func connectLiveReload(t *testing.T, serverURL string) *websocket.Conn {
	t.Helper()
	connection, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(serverURL, "http")+liveReloadWebSocketPath, nil)
	require.NoError(t, err)
	var hello liveReloadHello
	require.NoError(t, connection.ReadJSON(&hello))
	require.Equal(t, "hello", hello.Command)
	require.NoError(t, connection.WriteJSON(liveReloadClientHello{
		Command:   "hello",
		Protocols: []string{liveReloadProtocols[0]},
	}))
	return connection
}

func waitForLiveReloadConnection(t *testing.T, server *Server) {
	t.Helper()
	require.Eventually(t, func() bool {
		transport := server.currentLiveReloader()
		transport.mu.RLock()
		defer transport.mu.RUnlock()
		return len(transport.connections) == 1
	}, time.Second, time.Millisecond)
}
