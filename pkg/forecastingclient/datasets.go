package forecastingclient

import (
	"context"
	"encoding/json"
)

// UploadDataset sends a multipart/form-data POST to /admin/datasets/upload.
// fields: request_id, domain, source_mode, name, version, freq, scope, tenant, uniq
// file: the Excel/parquet file bytes, field name "file".
func (c *Client) UploadDataset(ctx context.Context, fields map[string]string, fileName string, fileBytes []byte) (*UploadDatasetResponse, error) {
	var resp UploadDatasetResponse
	err := c.uploadMultipart(ctx, "/admin/datasets/upload", fields, "file", fileName, fileBytes, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// PullPRL pulls a PRL dataset from ERP and registers it (POST /admin/datasets/pull/prl).
func (c *Client) PullPRL(ctx context.Context, req PullDatasetRequest) (*UploadDatasetResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	var resp UploadDatasetResponse
	// Pulling from ERP + building canonical series can be slow; use long timeout.
	if err := c.doLong(ctx, "POST", "/admin/datasets/pull/prl", body, "application/json", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// PullDN pulls a DN dataset from ERP and registers it (POST /admin/datasets/pull/dn).
func (c *Client) PullDN(ctx context.Context, req PullDatasetRequest) (*UploadDatasetResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	var resp UploadDatasetResponse
	if err := c.doLong(ctx, "POST", "/admin/datasets/pull/dn", body, "application/json", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetPRLBounds fetches the available PRL date bounds (GET /admin/datasets/pull/prl/bounds).
func (c *Client) GetPRLBounds(ctx context.Context, opts PullBoundsOptions) (*DatasetBoundsResponse, error) {
	path := "/admin/datasets/pull/prl/bounds"
	if q := buildQuery("scope", opts.Scope, "tenant", opts.Tenant, "uniq", opts.Uniq, "status", opts.Status); q != "" {
		path += "?" + q
	}
	var resp DatasetBoundsResponse
	if err := c.do(ctx, "GET", path, nil, "", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetDNBounds fetches the available DN date bounds (GET /admin/datasets/pull/dn/bounds).
func (c *Client) GetDNBounds(ctx context.Context, opts PullBoundsOptions) (*DatasetBoundsResponse, error) {
	path := "/admin/datasets/pull/dn/bounds"
	if q := buildQuery("scope", opts.Scope, "tenant", opts.Tenant, "uniq", opts.Uniq, "status", opts.Status); q != "" {
		path += "?" + q
	}
	var resp DatasetBoundsResponse
	if err := c.do(ctx, "GET", path, nil, "", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
