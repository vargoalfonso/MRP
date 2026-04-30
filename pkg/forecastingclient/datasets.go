package forecastingclient

import (
	"context"
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
