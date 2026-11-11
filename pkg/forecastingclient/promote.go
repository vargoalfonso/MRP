package forecastingclient

import (
	"context"
	"encoding/json"
)

// PromoteModel sends a promote request to /admin/promote.
func (c *Client) PromoteModel(ctx context.Context, req PromoteRequest) (*PromoteResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	var resp PromoteResponse
	err = c.do(ctx, "POST", "/admin/promote", body, "application/json", &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// ReloadActiveModel triggers /admin/reload?domain=<domain>.
func (c *Client) ReloadActiveModel(ctx context.Context, domain string) error {
	path := "/admin/reload"
	if domain != "" {
		path += "?domain=" + domain
	}
	return c.do(ctx, "POST", path, nil, "", nil)
}
