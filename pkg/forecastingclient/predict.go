package forecastingclient

import (
	"context"
	"encoding/json"
)

// Predict sends a forecast request to /predict.
func (c *Client) Predict(ctx context.Context, req PredictRequest) (*PredictResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	var resp PredictResponse
	err = c.do(ctx, "POST", "/predict", body, "application/json", &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
