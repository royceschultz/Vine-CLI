package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"vine/config"
	"vine/store"
)

// HealthResponse is returned by the /api/health endpoint.
type HealthResponse struct {
	Service   string `json:"service"`
	PID       int    `json:"pid"`
	StartedAt string `json:"started_at"`
}

// Server serves vine data over HTTP.
type Server struct {
	mux       *http.ServeMux
	startedAt time.Time
	Watcher   *Watcher
}

// New creates a new Server with all API routes registered.
func New() *Server {
	s := &Server{
		mux:       http.NewServeMux(),
		startedAt: time.Now().UTC(),
	}
	s.registerRoutes()
	return s
}

// SetWatcher attaches a file watcher and registers the WebSocket endpoint.
func (s *Server) SetWatcher(w *Watcher) {
	s.Watcher = w
	s.mux.HandleFunc("GET /api/projects/{project}/watch", w.HandleWatch)
}

// Handler returns the http.Handler for this server.
func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
	s.mux.HandleFunc("GET /api/projects", s.handleListProjects)
	s.mux.HandleFunc("GET /api/projects/{project}/tasks", s.handleListTasks)
	s.mux.HandleFunc("GET /api/projects/{project}/tasks/{id}", s.handleGetTask)
	s.mux.HandleFunc("GET /api/projects/{project}/tasks/{id}/children", s.handleChildTasks)
	s.mux.HandleFunc("GET /api/projects/{project}/tasks/{id}/comments", s.handleComments)
	s.mux.HandleFunc("GET /api/projects/{project}/tasks/{id}/dependencies", s.handleDependencies)
	s.mux.HandleFunc("GET /api/projects/{project}/tasks/{id}/dependents", s.handleDependents)
	s.mux.HandleFunc("GET /api/projects/{project}/tasks/{id}/tags", s.handleTaskTags)
	s.mux.HandleFunc("GET /api/projects/{project}/tasks/{id}/ancestors", s.handleAncestors)
	s.mux.HandleFunc("GET /api/projects/{project}/ready", s.handleReadyTasks)
	s.mux.HandleFunc("GET /api/projects/{project}/blocked", s.handleBlockedTasks)
	s.mux.HandleFunc("GET /api/projects/{project}/status", s.handleStatus)
	s.mux.HandleFunc("GET /api/projects/{project}/search", s.handleSearch)
	s.mux.HandleFunc("GET /api/projects/{project}/tags", s.handleListTags)
}

// openStore opens a store for the given global database name.
func openStore(project string) (*store.Store, error) {
	// Sanitize: only allow alphanumeric, dash, underscore, dot.
	for _, r := range project {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.') {
			return nil, fmt.Errorf("invalid project name: %q", project)
		}
	}

	dbDir, err := config.GlobalDatabasesDir()
	if err != nil {
		return nil, err
	}
	dbPath := filepath.Join(dbDir, project+".db")

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("project %q not found", project)
	}

	return store.OpenPath(dbPath)
}

// withStore is a helper that opens a store for the project path param,
// calls the handler, then closes the store.
func withStore(w http.ResponseWriter, r *http.Request, fn func(*store.Store) (any, error)) {
	project := r.PathValue("project")
	if project == "" {
		writeError(w, http.StatusBadRequest, "missing project name")
		return
	}

	s, err := openStore(project)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	defer s.Close()

	data, err := fn(s)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			writeError(w, http.StatusNotFound, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, data)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// --- Handlers ---

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, HealthResponse{
		Service:   "vine",
		PID:       os.Getpid(),
		StartedAt: s.startedAt.Format(time.RFC3339),
	})
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	dbDir, err := config.GlobalDatabasesDir()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	entries, err := os.ReadDir(dbDir)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusOK, []string{})
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var projects []string
	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() && strings.HasSuffix(name, ".db") && !strings.HasSuffix(name, "-wal") && !strings.HasSuffix(name, "-shm") {
			projects = append(projects, strings.TrimSuffix(name, ".db"))
		}
	}
	if projects == nil {
		projects = []string{}
	}

	writeJSON(w, http.StatusOK, projects)
}

func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	withStore(w, r, func(st *store.Store) (any, error) {
		filter := store.TaskFilter{
			Status:   r.URL.Query().Get("status"),
			Type:     r.URL.Query().Get("type"),
			Tag:      r.URL.Query().Get("tag"),
			All:      r.URL.Query().Get("all") == "true",
			RootOnly: r.URL.Query().Get("root") == "true",
		}
		return st.ListTasksFiltered(filter)
	})
}

func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	withStore(w, r, func(st *store.Store) (any, error) {
		return st.GetTask(r.PathValue("id"))
	})
}

func (s *Server) handleChildTasks(w http.ResponseWriter, r *http.Request) {
	withStore(w, r, func(st *store.Store) (any, error) {
		return st.ChildTasks(r.PathValue("id"))
	})
}

func (s *Server) handleComments(w http.ResponseWriter, r *http.Request) {
	withStore(w, r, func(st *store.Store) (any, error) {
		return st.CommentsForTask(r.PathValue("id"))
	})
}

func (s *Server) handleDependencies(w http.ResponseWriter, r *http.Request) {
	withStore(w, r, func(st *store.Store) (any, error) {
		return st.DependenciesOf(r.PathValue("id"))
	})
}

func (s *Server) handleDependents(w http.ResponseWriter, r *http.Request) {
	withStore(w, r, func(st *store.Store) (any, error) {
		return st.DependentsOf(r.PathValue("id"))
	})
}

func (s *Server) handleTaskTags(w http.ResponseWriter, r *http.Request) {
	withStore(w, r, func(st *store.Store) (any, error) {
		return st.TagsForTask(r.PathValue("id"))
	})
}

func (s *Server) handleReadyTasks(w http.ResponseWriter, r *http.Request) {
	withStore(w, r, func(st *store.Store) (any, error) {
		return st.ReadyTasks()
	})
}

func (s *Server) handleBlockedTasks(w http.ResponseWriter, r *http.Request) {
	withStore(w, r, func(st *store.Store) (any, error) {
		return st.BlockedTasks()
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	withStore(w, r, func(st *store.Store) (any, error) {
		detailed := r.URL.Query().Get("detailed") == "true"

		counts, err := st.TaskSummary()
		if err != nil {
			return nil, err
		}

		total := 0
		for _, c := range counts {
			total += c.Count
		}

		result := map[string]any{
			"project": r.PathValue("project"),
			"total":   total,
			"status":  counts,
		}

		if detailed {
			detailedCounts, err := st.TaskSummaryDetailed()
			if err != nil {
				return nil, err
			}
			result["detailed"] = detailedCounts
		}

		return result, nil
	})
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	withStore(w, r, func(st *store.Store) (any, error) {
		q := r.URL.Query().Get("q")
		if q == "" {
			return []store.Task{}, nil
		}
		return st.SearchTasks(q)
	})
}

func (s *Server) handleListTags(w http.ResponseWriter, r *http.Request) {
	withStore(w, r, func(st *store.Store) (any, error) {
		return st.ListTags()
	})
}

func (s *Server) handleAncestors(w http.ResponseWriter, r *http.Request) {
	withStore(w, r, func(st *store.Store) (any, error) {
		return st.AncestorChain(r.PathValue("id"))
	})
}
