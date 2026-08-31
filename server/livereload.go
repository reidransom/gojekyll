package server

import (
	"io"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/jaschaephraim/lrserver"
)

const (
	liveReloadScriptPath    = "/livereload.js"
	liveReloadWebSocketPath = "/livereload"
)

// liveReloadScriptTag is inserted into watched HTML pages. Its origin-relative
// URL lets the bundled client derive its endpoint from the page origin.
var liveReloadScriptTag = []byte(`<script src="/livereload.js"></script>`)

var liveReloadProtocols = []string{
	"http://livereload.com/protocols/official-7",
	"http://livereload.com/protocols/official-8",
	"http://livereload.com/protocols/official-9",
	"http://livereload.com/protocols/2.x-origin-version-negotiation",
	"http://livereload.com/protocols/2.x-remote-control",
}

// startLiveReloader makes the server-owned transport available without opening
// an auxiliary listener. It is safe to call more than once before serving.
func (s *Server) startLiveReloader() {
	s.m.Lock()
	defer s.m.Unlock()
	if s.liveReload == nil || s.liveReload.closed() {
		s.liveReload = newLiveReloadTransport()
	}
}

func (s *Server) stopLiveReloader() {
	s.m.Lock()
	liveReload := s.liveReload
	s.m.Unlock()
	if liveReload != nil {
		liveReload.Close()
	}
}

func (s *Server) currentLiveReloader() *liveReloadTransport {
	s.m.Lock()
	defer s.m.Unlock()
	return s.liveReload
}

// NewLiveReloadInjector returns a writer that injects the LiveReload client
// into its wrapped content.
func NewLiveReloadInjector(w io.Writer) io.Writer {
	return TagInjector{w, liveReloadScriptTag}
}

type liveReloadTransport struct {
	mu          sync.RWMutex
	connections map[*liveReloadConnection]struct{}
	isClosed    bool
}

func newLiveReloadTransport() *liveReloadTransport {
	return &liveReloadTransport{connections: make(map[*liveReloadConnection]struct{})}
}

func (t *liveReloadTransport) closed() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.isClosed
}

func (t *liveReloadTransport) ServeScript(rw http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet {
		rw.WriteHeader(http.StatusMethodNotAllowed)
		return nil
	}
	rw.Header().Set("Content-Type", "application/javascript")
	_, err := io.WriteString(rw, lrserver.JS)
	return err
}

func (t *liveReloadTransport) ServeWebSocket(rw http.ResponseWriter, r *http.Request) {
	connection, err := (websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}).Upgrade(rw, r, nil)
	if err != nil {
		return
	}

	connection.SetReadLimit(1 << 20)
	if err := connection.WriteJSON(liveReloadHello{Command: "hello", Protocols: liveReloadProtocols, ServerName: lrserver.DefaultName}); err != nil {
		_ = connection.Close()
		return
	}

	var hello liveReloadClientHello
	if err := connection.ReadJSON(&hello); err != nil || !validLiveReloadHello(hello) {
		_ = connection.Close()
		return
	}

	client := &liveReloadConnection{
		connection: connection,
		transport:  t,
		outgoing:   make(chan interface{}, 16),
		done:       make(chan struct{}),
	}
	if !t.add(client) {
		_ = connection.Close()
		return
	}
	go client.read()
	go client.write()
}

func (t *liveReloadTransport) add(connection *liveReloadConnection) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.isClosed {
		return false
	}
	t.connections[connection] = struct{}{}
	return true
}

func (t *liveReloadTransport) remove(connection *liveReloadConnection) {
	t.mu.Lock()
	delete(t.connections, connection)
	t.mu.Unlock()
}

func (t *liveReloadTransport) Reload(path string) {
	t.broadcast(liveReloadReload{Command: "reload", Path: path, LiveCSS: true})
}

// Deliver broadcasts the work described by one watch batch.
func (t *liveReloadTransport) Deliver(intent liveReloadIntent) {
	if intent.pageReload {
		t.Reload("")
		return
	}
	for _, path := range intent.resourcePaths {
		t.Reload(path)
	}
}

func (t *liveReloadTransport) Alert(message string) {
	t.broadcast(liveReloadAlert{Command: "alert", Message: message})
}

func (t *liveReloadTransport) broadcast(message interface{}) {
	t.mu.RLock()
	if t.isClosed {
		t.mu.RUnlock()
		return
	}
	connections := make([]*liveReloadConnection, 0, len(t.connections))
	for connection := range t.connections {
		connections = append(connections, connection)
	}
	t.mu.RUnlock()

	for _, connection := range connections {
		connection.send(message)
	}
}

func (t *liveReloadTransport) Close() {
	t.mu.Lock()
	if t.isClosed {
		t.mu.Unlock()
		return
	}
	t.isClosed = true
	connections := make([]*liveReloadConnection, 0, len(t.connections))
	for connection := range t.connections {
		connections = append(connections, connection)
	}
	t.mu.Unlock()

	for _, connection := range connections {
		connection.close()
	}
}

type liveReloadConnection struct {
	connection *websocket.Conn
	transport  *liveReloadTransport
	outgoing   chan interface{}
	done       chan struct{}
	closeOnce  sync.Once
}

func (c *liveReloadConnection) read() {
	defer c.close()
	for {
		var message interface{}
		if err := c.connection.ReadJSON(&message); err != nil {
			return
		}
	}
}

func (c *liveReloadConnection) write() {
	for {
		select {
		case message := <-c.outgoing:
			if err := c.connection.WriteJSON(message); err != nil {
				c.close()
				return
			}
		case <-c.done:
			return
		}
	}
}

func (c *liveReloadConnection) send(message interface{}) {
	select {
	case <-c.done:
		return
	case c.outgoing <- message:
	default:
		c.close()
	}
}

func (c *liveReloadConnection) close() {
	c.closeOnce.Do(func() {
		close(c.done)
		c.transport.remove(c)
		_ = c.connection.Close()
	})
}

type liveReloadClientHello struct {
	Command   string   `json:"command"`
	Protocols []string `json:"protocols"`
}

type liveReloadHello struct {
	Command    string   `json:"command"`
	Protocols  []string `json:"protocols"`
	ServerName string   `json:"serverName"`
}

type liveReloadReload struct {
	Command string `json:"command"`
	Path    string `json:"path"`
	LiveCSS bool   `json:"liveCSS"`
}

type liveReloadAlert struct {
	Command string `json:"command"`
	Message string `json:"message"`
}

func validLiveReloadHello(hello liveReloadClientHello) bool {
	if hello.Command != "hello" {
		return false
	}
	for _, protocol := range hello.Protocols {
		for _, supported := range liveReloadProtocols {
			if protocol == supported {
				return true
			}
		}
	}
	return false
}
