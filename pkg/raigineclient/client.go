package raigineclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/ganasa18/go-template/pkg/apperror"
)

const defaultTimeout = 30 * time.Second

// Client wraps HTTP calls to the Raigine automation platform (crp-backend).
type Client struct {
	baseURL     string
	staticToken string
	email       string
	password    string
	httpClient  *http.Client

	mu          sync.Mutex
	cachedToken string
	tokenExpiry time.Time
}

// New builds a Raigine client from Options.
func New(opts Options) *Client {
	timeout := defaultTimeout
	if opts.TimeoutSeconds > 0 {
		timeout = time.Duration(opts.TimeoutSeconds) * time.Second
	}
	return &Client{
		baseURL:     opts.BaseURL,
		staticToken: opts.StaticToken,
		email:       opts.Email,
		password:    opts.Password,
		httpClient:  &http.Client{Timeout: timeout},
	}
}

// Enabled reports whether the client has enough configuration to talk to crp-backend.
func (c *Client) Enabled() bool {
	return c != nil && c.baseURL != "" && (c.staticToken != "" || (c.email != "" && c.password != ""))
}

// token returns a valid bearer token, logging in and caching it when needed.
func (c *Client) token(ctx context.Context) (string, error) {
	if c.staticToken != "" {
		return c.staticToken, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cachedToken != "" && time.Now().Before(c.tokenExpiry) {
		return c.cachedToken, nil
	}

	if c.email == "" || c.password == "" {
		return "", apperror.Internal("raigine client not configured: missing token or credentials")
	}

	body, _ := json.Marshal(loginRequest{Email: c.email, Password: c.password})
	var resp loginResponse
	if err := c.do(ctx, http.MethodPost, "/api/v1/auth/login", body, "", &resp); err != nil {
		return "", err
	}
	if resp.Data.Access.Token == "" {
		return "", apperror.Internal("raigine login returned an empty token")
	}

	c.cachedToken = resp.Data.Access.Token
	// Refresh a bit before the real expiry; fall back to 50 minutes.
	if t, err := time.Parse(time.RFC3339, resp.Data.Access.ExpiresAt); err == nil {
		c.tokenExpiry = t.Add(-1 * time.Minute)
	} else {
		c.tokenExpiry = time.Now().Add(50 * time.Minute)
	}
	return c.cachedToken, nil
}

// do is the low-level HTTP helper. When authToken is non-empty it is sent as a
// bearer token; pass "" for unauthenticated calls (e.g. login).
func (c *Client) do(ctx context.Context, method, path string, bodyBytes []byte, authToken string, dest interface{}) error {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader(bodyBytes))
	if err != nil {
		return apperror.InternalWrap("build raigine request failed", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return apperror.GatewayTimeout("raigine automation platform unreachable", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20)) // 2 MB limit
	if err != nil {
		return apperror.InternalWrap("read raigine response failed", err)
	}

	if resp.StatusCode >= 400 {
		var apiErr APIErrorResponse
		if json.Unmarshal(respBody, &apiErr) == nil && apiErr.Message != "" {
			return apperror.New(resp.StatusCode, apperror.CodeInternalError, apiErr.Message)
		}
		return apperror.New(resp.StatusCode, apperror.CodeInternalError,
			fmt.Sprintf("raigine API error %d: %s", resp.StatusCode, string(respBody)))
	}

	if dest == nil {
		return nil
	}
	if len(respBody) == 0 {
		return nil
	}
	if err := json.Unmarshal(respBody, dest); err != nil {
		return apperror.InternalWrap("decode raigine response failed", err)
	}
	return nil
}

func bodyReader(b []byte) io.Reader {
	if len(b) == 0 {
		return nil
	}
	return bytes.NewReader(b)
}

// authed runs a request with a valid bearer token.
func (c *Client) authed(ctx context.Context, method, path string, bodyBytes []byte, dest interface{}) error {
	token, err := c.token(ctx)
	if err != nil {
		return err
	}
	return c.do(ctx, method, path, bodyBytes, token, dest)
}

// RunProcess triggers an automation process by its public id.
// crp-backend routes it to a Tower agent or Cloud Run based on the process's
// executionMode.
func (c *Client) RunProcess(ctx context.Context, processPublicID string, req RunProcessRequest) (map[string]interface{}, error) {
	body, _ := json.Marshal(req)
	var out map[string]interface{}
	err := c.authed(ctx, http.MethodPost,
		fmt.Sprintf("/api/v1/automation-process/%s/run", url.PathEscape(processPublicID)), body, &out)
	return out, err
}

// StopProcess stops a running automation process by its public id.
func (c *Client) StopProcess(ctx context.Context, processPublicID string) (map[string]interface{}, error) {
	var out map[string]interface{}
	err := c.authed(ctx, http.MethodPost,
		fmt.Sprintf("/api/v1/automation-process/%s/stop", url.PathEscape(processPublicID)), nil, &out)
	return out, err
}

// ListProcesses returns paginated automation processes for the caller's org.
func (c *Client) ListProcesses(ctx context.Context, p ListParams) (map[string]interface{}, error) {
	q := url.Values{}
	if p.Page > 0 {
		q.Set("page", strconv.Itoa(p.Page))
	}
	if p.Limit > 0 {
		q.Set("limit", strconv.Itoa(p.Limit))
	}
	if p.Pagination != "" {
		q.Set("pagination", p.Pagination)
	}
	if p.FolderID != "" {
		q.Set("folderId", p.FolderID)
	}
	if p.MachineID != "" {
		q.Set("machineId", p.MachineID)
	}
	var out map[string]interface{}
	err := c.authed(ctx, http.MethodGet, "/api/v1/automation-process?"+q.Encode(), nil, &out)
	return out, err
}

// ListJobs returns paginated automation job history for the caller's org.
func (c *Client) ListJobs(ctx context.Context, p ListParams) (map[string]interface{}, error) {
	q := url.Values{}
	if p.Page > 0 {
		q.Set("page", strconv.Itoa(p.Page))
	}
	if p.Limit > 0 {
		q.Set("limit", strconv.Itoa(p.Limit))
	}
	if p.ProcessID != "" {
		q.Set("processId", p.ProcessID)
	}
	if p.FolderID != "" {
		q.Set("folderId", p.FolderID)
	}
	if p.MachineID != "" {
		q.Set("machineId", p.MachineID)
	}
	var out map[string]interface{}
	err := c.authed(ctx, http.MethodGet, "/api/v1/automation-jobs?"+q.Encode(), nil, &out)
	return out, err
}

// CreateSchedule registers a cron schedule on crp-backend.
func (c *Client) CreateSchedule(ctx context.Context, req CreateScheduleRequest) (map[string]interface{}, error) {
	body, _ := json.Marshal(req)
	var out map[string]interface{}
	err := c.authed(ctx, http.MethodPost, "/api/v1/automation-schedules", body, &out)
	return out, err
}
