package raigineclient

// This package wraps HTTP calls from the MRP backend to the Raigine
// automation platform (crp-backend). MRP is the SOURCE of business events;
// crp-backend is the ORCHESTRATOR that decides whether an automation runs on a
// Tower agent (executionMode = "local") or on Cloud Run (executionMode =
// "cloud_run"). MRP therefore integrates with crp-backend only, never with
// Tower directly.

// Options configures the Raigine client.
type Options struct {
	// BaseURL of crp-backend, e.g. http://localhost:8080 (WITHOUT trailing slash).
	BaseURL string

	// StaticToken, when set, is sent as `Authorization: Bearer <token>` on every
	// request and the login flow is skipped. Use this for a long-lived service
	// token / machine token.
	StaticToken string

	// Email/Password are used to obtain (and refresh) an access token via
	// POST /api/v1/auth/login when StaticToken is empty.
	Email    string
	Password string

	// TimeoutSeconds is the per-request timeout. Defaults to 30s when 0.
	TimeoutSeconds int
}

// --- Auth ---

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Data struct {
		Access struct {
			Token     string `json:"token"`
			ExpiresAt string `json:"expires_at"`
		} `json:"access"`
		Refresh struct {
			Token     string `json:"token"`
			ExpiresAt string `json:"expires_at"`
		} `json:"refresh"`
	} `json:"data"`
}

// --- Automation Process ---

// RunProcessRequest is the body for POST /automation-process/:id/run.
type RunProcessRequest struct {
	// JobPublicID is optional; when empty crp-backend creates a new job.
	JobPublicID string `json:"job_public_id,omitempty"`
}

// ListParams are the common pagination/filter query params.
type ListParams struct {
	Page       int
	Limit      int
	Pagination string // "true" | "false"
	FolderID   string
	MachineID  string
	ProcessID  string
}

// --- Automation Schedule ---

// CreateScheduleRequest maps to crp-backend automationScheduleSchema
// (POST /api/v1/automation-schedules).
type CreateScheduleRequest struct {
	ScheduleName        string          `json:"scheduleName"`
	AutomationProcessID string          `json:"automationProcessId"`
	TriggerPriority     string          `json:"triggerPriority,omitempty"` // Low | Medium | High
	ExecutionFrequency  string          `json:"executionFrequency"`        // every_minute | hourly | daily | weekly | monthly
	ExecuteTimes        int             `json:"executeTimes,omitempty"`
	StartTime           string          `json:"startTime"`
	Timezone            string          `json:"timezone,omitempty"`
	FolderID            string          `json:"folderId,omitempty"`
	RepeatInterval      string          `json:"repeatInterval,omitempty"`
	DaysOfWeek          map[string]bool `json:"daysOfWeek,omitempty"`
	DaysOfMonth         string          `json:"daysOfMonth,omitempty"`
	AutoDisable         bool            `json:"autoDisable"`
	DisableDate         string          `json:"disableDate,omitempty"`
	AutoEnd             bool            `json:"autoEnd"`
	EndDate             string          `json:"endDate,omitempty"`
	AlertIfStuck        bool            `json:"alertIfStuck"`
	IsActive            *bool           `json:"isActive,omitempty"`
}

// APIErrorResponse is the shape crp-backend returns on 4xx/5xx.
type APIErrorResponse struct {
	Message string `json:"message"`
	Error   string `json:"error"`
}
