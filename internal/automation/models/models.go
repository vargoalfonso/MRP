package models

// RunRequest is the body accepted by MRP's run endpoint. All fields optional.
type RunRequest struct {
	JobPublicID string `json:"job_public_id,omitempty"`
}

// ScheduleRequest is the body accepted by MRP's create-schedule endpoint.
// It mirrors the Raigine automationScheduleSchema but keeps MRP-friendly names.
type ScheduleRequest struct {
	ScheduleName        string          `json:"schedule_name" validate:"required"`
	AutomationProcessID string          `json:"automation_process_id" validate:"required"`
	TriggerPriority     string          `json:"trigger_priority,omitempty"`
	ExecutionFrequency  string          `json:"execution_frequency" validate:"required"`
	ExecuteTimes        int             `json:"execute_times,omitempty"`
	StartTime           string          `json:"start_time" validate:"required"`
	Timezone            string          `json:"timezone,omitempty"`
	FolderID            string          `json:"folder_id,omitempty"`
	RepeatInterval      string          `json:"repeat_interval,omitempty"`
	DaysOfWeek          map[string]bool `json:"days_of_week,omitempty"`
	DaysOfMonth         string          `json:"days_of_month,omitempty"`
	AutoDisable         bool            `json:"auto_disable,omitempty"`
	DisableDate         string          `json:"disable_date,omitempty"`
	AutoEnd             bool            `json:"auto_end,omitempty"`
	EndDate             string          `json:"end_date,omitempty"`
	AlertIfStuck        bool            `json:"alert_if_stuck,omitempty"`
}
