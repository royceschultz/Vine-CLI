package server

import (
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"

	"vine/config"
)

// WatchEvent is sent to WebSocket clients when a project's data changes.
type WatchEvent struct {
	Type      string `json:"type"`
	Project   string `json:"project"`
	Timestamp string `json:"timestamp"`
}

// Watcher monitors SQLite database files for changes and broadcasts
// events to connected WebSocket clients. The file watcher is only
// active when at least one client is connected.
type Watcher struct {
	mu      sync.Mutex
	clients map[string]map[*websocket.Conn]struct{} // project -> connections
	total   int                                      // total connected clients
	logger  *log.Logger

	// fsnotify lifecycle
	fsWatcher *fsnotify.Watcher
	fsDone    chan struct{}

	// debounce per project
	debounce map[string]*time.Timer
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// NewWatcher creates a Watcher.
func NewWatcher(logger *log.Logger) *Watcher {
	return &Watcher{
		clients:  make(map[string]map[*websocket.Conn]struct{}),
		debounce: make(map[string]*time.Timer),
		logger:   logger,
	}
}

// Stop tears down the file watcher if running.
func (w *Watcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.stopFsWatcherLocked()
}

// startFsWatcherLocked starts fsnotify. Caller must hold w.mu.
func (w *Watcher) startFsWatcherLocked() {
	if w.fsWatcher != nil {
		return // already running
	}

	dbDir, err := config.GlobalDatabasesDir()
	if err != nil {
		w.logger.Printf("watcher: cannot resolve database dir: %v", err)
		return
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		w.logger.Printf("watcher: failed to create fsnotify watcher: %v", err)
		return
	}

	if err := watcher.Add(dbDir); err != nil {
		w.logger.Printf("watcher: failed to watch %s: %v", dbDir, err)
		watcher.Close()
		return
	}

	w.fsWatcher = watcher
	w.fsDone = make(chan struct{})

	w.logger.Printf("watcher: started monitoring %s", dbDir)

	go w.fsLoop(watcher, w.fsDone)
}

// stopFsWatcherLocked stops fsnotify. Caller must hold w.mu.
func (w *Watcher) stopFsWatcherLocked() {
	if w.fsWatcher == nil {
		return
	}
	close(w.fsDone)
	w.fsWatcher.Close()
	w.fsWatcher = nil
	w.fsDone = nil
	w.logger.Printf("watcher: stopped monitoring (no clients)")
}

func (w *Watcher) fsLoop(watcher *fsnotify.Watcher, done <-chan struct{}) {
	for {
		select {
		case <-done:
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				w.handleFileEvent(event.Name)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			w.logger.Printf("watcher error: %v", err)
		}
	}
}

func (w *Watcher) handleFileEvent(path string) {
	name := filepath.Base(path)

	// Only care about .db files (not WAL/SHM).
	if !strings.HasSuffix(name, ".db") {
		return
	}

	project := strings.TrimSuffix(name, ".db")

	w.mu.Lock()
	if timer, ok := w.debounce[project]; ok {
		timer.Stop()
	}
	w.debounce[project] = time.AfterFunc(100*time.Millisecond, func() {
		w.broadcast(project)
	})
	w.mu.Unlock()
}

func (w *Watcher) broadcast(project string) {
	event := WatchEvent{
		Type:      "changed",
		Project:   project,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.Marshal(event)
	if err != nil {
		w.logger.Printf("watcher: marshal error: %v", err)
		return
	}

	w.mu.Lock()
	conns := w.clients[project]
	if len(conns) == 0 {
		w.mu.Unlock()
		return
	}

	// Copy the set so we can release the lock before writing.
	snapshot := make([]*websocket.Conn, 0, len(conns))
	for conn := range conns {
		snapshot = append(snapshot, conn)
	}
	w.mu.Unlock()

	w.logger.Printf("watcher: broadcasting change for %q to %d client(s)", project, len(snapshot))

	var dead []*websocket.Conn
	for _, conn := range snapshot {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			dead = append(dead, conn)
		}
	}

	if len(dead) > 0 {
		w.mu.Lock()
		for _, conn := range dead {
			delete(w.clients[project], conn)
			conn.Close()
			w.total--
		}
		if w.total == 0 {
			w.stopFsWatcherLocked()
		}
		w.mu.Unlock()
	}
}

// subscribe adds a WebSocket connection to the watch list for a project.
// Starts the file watcher if this is the first client.
func (w *Watcher) subscribe(project string, conn *websocket.Conn) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.clients[project] == nil {
		w.clients[project] = make(map[*websocket.Conn]struct{})
	}
	w.clients[project][conn] = struct{}{}
	w.total++
	w.logger.Printf("watcher: client subscribed to %q (%d total)", project, w.total)

	// Start file watcher on first client.
	if w.total == 1 {
		w.startFsWatcherLocked()
	}
}

// unsubscribe removes a WebSocket connection.
// Stops the file watcher if this was the last client.
func (w *Watcher) unsubscribe(project string, conn *websocket.Conn) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if conns, ok := w.clients[project]; ok {
		if _, exists := conns[conn]; exists {
			delete(conns, conn)
			w.total--
		}
		if len(conns) == 0 {
			delete(w.clients, project)
		}
	}
	w.logger.Printf("watcher: client unsubscribed from %q (%d total)", project, w.total)

	// Stop file watcher when no clients remain.
	if w.total == 0 {
		w.stopFsWatcherLocked()
	}
}

// HandleWatch is the HTTP handler for the WebSocket watch endpoint.
func (w *Watcher) HandleWatch(wr http.ResponseWriter, r *http.Request) {
	project := r.PathValue("project")
	if project == "" {
		writeError(wr, http.StatusBadRequest, "missing project name")
		return
	}

	conn, err := upgrader.Upgrade(wr, r, nil)
	if err != nil {
		w.logger.Printf("watcher: upgrade error: %v", err)
		return
	}
	defer conn.Close()

	w.subscribe(project, conn)
	defer w.unsubscribe(project, conn)

	// Keep the connection alive by reading (and discarding) client messages.
	// The connection closes when the client disconnects or an error occurs.
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}
