package forecastingclient

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
)

// GetTrainingRun fetches a single training run by its external training_run_id.
func (c *Client) GetTrainingRun(ctx context.Context, trainingRunID string) (*TrainingRunDetail, error) {
	var resp TrainingRunDetail
	err := c.do(ctx, "GET", "/admin/training-runs/"+trainingRunID, nil, "", &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListTrainingRunsOptions defines filter params for listing training runs.
type ListTrainingRunsOptions struct {
	Scope   string
	Tenant  string
	Uniq    string
	Status  string
	Limit   int
}

// ListTrainingRuns queries /admin/training-runs with optional filters.
func (c *Client) ListTrainingRuns(ctx context.Context, opts ListTrainingRunsOptions) ([]TrainingRunListItem, error) {
	q := url.Values{}
	if opts.Scope != "" {
		q.Set("scope", opts.Scope)
	}
	if opts.Tenant != "" {
		q.Set("tenant", opts.Tenant)
	}
	if opts.Uniq != "" {
		q.Set("uniq", opts.Uniq)
	}
	if opts.Status != "" {
		q.Set("status", opts.Status)
	}
	if opts.Limit > 0 {
		q.Set("limit", string(rune(opts.Limit)))
	}
	path := "/admin/training-runs"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var resp []TrainingRunListItem
	err := c.do(ctx, "GET", path, nil, "", &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// ListDatasets queries /admin/datasets.
func (c *Client) ListDatasets(ctx context.Context, name string, limit int) ([]DatasetListItem, error) {
	q := url.Values{}
	if name != "" {
		q.Set("name", name)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	path := "/admin/datasets"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var resp []DatasetListItem
	err := c.do(ctx, "GET", path, nil, "", &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// ListModelVersionsOptions defines filter params for model versions.
type ListModelVersionsOptions struct {
	Scope   string
	Tenant  string
	Uniq    string
	Status  string
	Limit   int
}

// ListModelVersions queries /admin/model-versions.
func (c *Client) ListModelVersions(ctx context.Context, opts ListModelVersionsOptions) ([]ModelVersionItem, error) {
	q := url.Values{}
	if opts.Scope != "" {
		q.Set("scope", opts.Scope)
	}
	if opts.Tenant != "" {
		q.Set("tenant", opts.Tenant)
	}
	if opts.Uniq != "" {
		q.Set("uniq", opts.Uniq)
	}
	if opts.Status != "" {
		q.Set("status", opts.Status)
	}
	if opts.Limit > 0 {
		q.Set("limit", string(rune(opts.Limit)))
	}
	path := "/admin/model-versions"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var resp []ModelVersionItem
	err := c.do(ctx, "GET", path, nil, "", &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// ListDeploymentsOptions defines filter params for deployments.
type ListDeploymentsOptions struct {
	Stage   string
	Scope   string
	Tenant  string
	Uniq    string
	Limit   int
}

// ListDeployments queries /admin/deployments.
func (c *Client) ListDeployments(ctx context.Context, opts ListDeploymentsOptions) ([]DeploymentItem, error) {
	q := url.Values{}
	if opts.Stage != "" {
		q.Set("stage", opts.Stage)
	}
	if opts.Scope != "" {
		q.Set("scope", opts.Scope)
	}
	if opts.Tenant != "" {
		q.Set("tenant", opts.Tenant)
	}
	if opts.Uniq != "" {
		q.Set("uniq", opts.Uniq)
	}
	if opts.Limit > 0 {
		q.Set("limit", string(rune(opts.Limit)))
	}
	path := "/admin/deployments"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var resp []DeploymentItem
	err := c.do(ctx, "GET", path, nil, "", &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetInferenceResult fetches a single inference result by request_id.
func (c *Client) GetInferenceResult(ctx context.Context, requestID string) (*InferenceResultDetail, error) {
	var resp InferenceResultDetail
	err := c.do(ctx, "GET", "/admin/inference-results/"+requestID, nil, "", &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListInferenceResultsOptions defines filter params for inference results.
type ListInferenceResultsOptions struct {
	RequestID string
	Status    string
	Limit     int
}

// ListInferenceResults queries /admin/inference-results.
func (c *Client) ListInferenceResults(ctx context.Context, opts ListInferenceResultsOptions) ([]InferenceResultItem, error) {
	q := url.Values{}
	if opts.RequestID != "" {
		q.Set("request_id", opts.RequestID)
	}
	if opts.Status != "" {
		q.Set("status", opts.Status)
	}
	if opts.Limit > 0 {
		q.Set("limit", string(rune(opts.Limit)))
	}
	path := "/admin/inference-results"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var resp []InferenceResultItem
	err := c.do(ctx, "GET", path, nil, "", &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// RawGet performs a raw GET to an arbitrary path and returns the raw JSON bytes.
// Used internally for fallback when structured response parsing is not needed.
func (c *Client) RawGet(ctx context.Context, path string) (json.RawMessage, error) {
	var raw json.RawMessage
	err := c.do(ctx, "GET", path, nil, "", &raw)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

// buildQuery builds a URL query string from key-value pairs, skipping empty values.
func buildQuery(pairs ...string) string {
	var parts []string
	for i := 0; i < len(pairs)-1; i += 2 {
		if pairs[i+1] != "" {
			parts = append(parts, pairs[i]+"="+url.QueryEscape(pairs[i+1]))
		}
	}
	return strings.Join(parts, "&")
}
