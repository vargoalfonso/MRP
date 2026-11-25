package forecastingclient

import "context"

// SchedulerStatus is one item of GET /admin/scheduler/status: the latest run of a
// domain (dn, prl) that was started by the forecasting scheduler. Domains that
// the scheduler never ran are not returned.
type SchedulerStatus struct {
	Domain string `json:"domain"`
	// LastRunAt: finish time; if the run is not finished yet, its start (or
	// creation) time. RFC3339, UTC.
	LastRunAt *string `json:"last_run_at"`
	// LastRunStatus: RUNNING | SUCCEEDED | FAILED (cancelled counts as FAILED).
	LastRunStatus *string `json:"last_run_status"`
	// LastRunMessage: error message of the training run, or nil.
	LastRunMessage *string `json:"last_run_message"`
	// LastTrainingRunID: training run id (see GET /forecasting/training-runs/:id).
	LastTrainingRunID *string `json:"last_training_run_id"`
}

// ListSchedulerStatus queries GET /admin/scheduler/status.
//
// The response is a bare JSON array (no envelope). The endpoint only reads the
// forecasting DB and never calls the ERP back, so it is safe to call from inside
// the ERP's own schedule listing.
func (c *Client) ListSchedulerStatus(ctx context.Context) ([]SchedulerStatus, error) {
	out := make([]SchedulerStatus, 0)
	if err := c.do(ctx, "GET", "/admin/scheduler/status", nil, "", &out); err != nil {
		return nil, err
	}
	if out == nil { // a JSON null body
		out = make([]SchedulerStatus, 0)
	}
	return out, nil
}
