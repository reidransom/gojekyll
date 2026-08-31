package server

import (
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

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

func requireNoLiveReloadMessage(t *testing.T, connection *websocket.Conn) {
	t.Helper()
	require.NoError(t, connection.SetReadDeadline(time.Now().Add(100*time.Millisecond)))
	var message liveReloadReload
	require.Error(t, connection.ReadJSON(&message))
	require.NoError(t, connection.SetReadDeadline(time.Time{}))
}
