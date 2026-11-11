package forecastingclient

import (
	"context"
	"encoding/json"
)

// TrainGlobal triggers a global domain training run.
func (c *Client) TrainGlobal(ctx context.Context, req TrainGlobalRequest) (*TrainResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	var resp TrainResponse
	err = c.doLong(ctx, "POST", "/admin/train/global", body, "application/json", &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// TrainCustom triggers a custom (per-tenant/per-uniq) training run.
func (c *Client) TrainCustom(ctx context.Context, req TrainCustomRequest) (*TrainResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	var resp TrainResponse
	err = c.doLong(ctx, "POST", "/admin/train/custom", body, "application/json", &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
