package client

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"vine/config"
	"vine/server"
	"vine/store"
)

// Client queries a remote vine server over HTTP, optionally through an SSH tunnel.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// New creates a Client from a remote config. For SSH remotes, it connects
// through a persistent SSH tunnel (auto-connecting if needed).
func New(remote *config.Remote) (*Client, error) {
	if remote.IsSSH() {
		return newSSHClient(remote)
	}
	return newHTTPClient(remote), nil
}

// newHTTPClient creates a direct HTTP client (opt-in via --http).
func newHTTPClient(remote *config.Remote) *Client {
	transport := &http.Transport{}
	if remote.TLS {
		transport.TLSClientConfig = &tls.Config{}
	}

	return &Client{
		baseURL: remote.URL(),
		token:   remote.Token,
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
	}
}

// newSSHClient reuses a persistent SSH tunnel or auto-connects one.
func newSSHClient(remote *config.Remote) (*Client, error) {
	info, err := Connect(remote)
	if err != nil {
		return nil, err
	}

	return &Client{
		baseURL: fmt.Sprintf("http://127.0.0.1:%d", info.LocalPort),
		token:   remote.Token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// Close is a no-op for persistent tunnels. Use Disconnect() to tear down tunnels.
func (c *Client) Close() {}

func (c *Client) get(path string, query url.Values, result any) error {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("remote unreachable (is the tunnel still connected?): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		if json.NewDecoder(resp.Body).Decode(&errResp) == nil && errResp.Error != "" {
			return fmt.Errorf("remote error (%d): %s", resp.StatusCode, errResp.Error)
		}
		return fmt.Errorf("remote returned status %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	return nil
}

// Health checks the remote server health.
func (c *Client) Health() (*server.HealthResponse, error) {
	var health server.HealthResponse
	if err := c.get("/api/health", nil, &health); err != nil {
		return nil, err
	}
	return &health, nil
}

// ListProjects returns the list of global database names on the remote.
func (c *Client) ListProjects() ([]string, error) {
	var projects []string
	if err := c.get("/api/projects", nil, &projects); err != nil {
		return nil, err
	}
	return projects, nil
}

// ListTasks returns tasks for a project with optional filtering.
func (c *Client) ListTasks(project string, filter store.TaskFilter) ([]store.TaskWithDeps, error) {
	q := url.Values{}
	if filter.Status != "" {
		q.Set("status", filter.Status)
	}
	if filter.Type != "" {
		q.Set("type", filter.Type)
	}
	if filter.Tag != "" {
		q.Set("tag", filter.Tag)
	}
	if filter.All {
		q.Set("all", "true")
	}
	if filter.RootOnly {
		q.Set("root", "true")
	}

	var tasks []store.TaskWithDeps
	if err := c.get("/api/projects/"+project+"/tasks", q, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

// GetTask returns a single task.
func (c *Client) GetTask(project, id string) (*store.Task, error) {
	var task store.Task
	if err := c.get("/api/projects/"+project+"/tasks/"+id, nil, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

// ChildTasks returns children of a task.
func (c *Client) ChildTasks(project, parentID string) ([]store.Task, error) {
	var tasks []store.Task
	if err := c.get("/api/projects/"+project+"/tasks/"+parentID+"/children", nil, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

// AncestorChain returns the parent chain from immediate parent up to root.
func (c *Client) AncestorChain(project, taskID string) ([]store.Task, error) {
	var tasks []store.Task
	if err := c.get("/api/projects/"+project+"/tasks/"+taskID+"/ancestors", nil, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

// CommentsForTask returns comments on a task.
func (c *Client) CommentsForTask(project, taskID string) ([]store.Comment, error) {
	var comments []store.Comment
	if err := c.get("/api/projects/"+project+"/tasks/"+taskID+"/comments", nil, &comments); err != nil {
		return nil, err
	}
	return comments, nil
}

// DependenciesOf returns what a task depends on.
func (c *Client) DependenciesOf(project, taskID string) ([]store.Dependency, error) {
	var deps []store.Dependency
	if err := c.get("/api/projects/"+project+"/tasks/"+taskID+"/dependencies", nil, &deps); err != nil {
		return nil, err
	}
	return deps, nil
}

// DependentsOf returns what depends on a task.
func (c *Client) DependentsOf(project, taskID string) ([]store.Dependency, error) {
	var deps []store.Dependency
	if err := c.get("/api/projects/"+project+"/tasks/"+taskID+"/dependents", nil, &deps); err != nil {
		return nil, err
	}
	return deps, nil
}

// TagsForTask returns tags on a task.
func (c *Client) TagsForTask(project, taskID string) ([]store.Tag, error) {
	var tags []store.Tag
	if err := c.get("/api/projects/"+project+"/tasks/"+taskID+"/tags", nil, &tags); err != nil {
		return nil, err
	}
	return tags, nil
}

// ReadyTasks returns tasks that are ready to work on.
func (c *Client) ReadyTasks(project string) ([]store.Task, error) {
	var tasks []store.Task
	if err := c.get("/api/projects/"+project+"/ready", nil, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

// BlockedTasks returns tasks that are blocked.
func (c *Client) BlockedTasks(project string) ([]store.Task, error) {
	var tasks []store.Task
	if err := c.get("/api/projects/"+project+"/blocked", nil, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

// StatusResponse mirrors the server's status endpoint response.
type StatusResponse struct {
	Project  string              `json:"project"`
	Total    int                 `json:"total"`
	Status   []store.StatusCount `json:"status"`
	Detailed []store.StatusTypeCount `json:"detailed,omitempty"`
}

// Status returns the task summary for a project.
func (c *Client) Status(project string, detailed bool) (*StatusResponse, error) {
	q := url.Values{}
	if detailed {
		q.Set("detailed", "true")
	}
	var resp StatusResponse
	if err := c.get("/api/projects/"+project+"/status", q, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// SearchTasks searches tasks by keyword.
func (c *Client) SearchTasks(project, query string) ([]store.Task, error) {
	q := url.Values{}
	q.Set("q", query)
	var tasks []store.Task
	if err := c.get("/api/projects/"+project+"/search", q, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

// ListTags returns all tags with counts.
func (c *Client) ListTags(project string) ([]store.TagWithCount, error) {
	var tags []store.TagWithCount
	if err := c.get("/api/projects/"+project+"/tags", nil, &tags); err != nil {
		return nil, err
	}
	return tags, nil
}
