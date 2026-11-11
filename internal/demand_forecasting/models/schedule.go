package models

import "time"

// ForecastingSchedule mirrors the forecasting_schedules table (max 1 row per domain).
// JSON tags match the frontend type ForecastSchedule.
type ForecastingSchedule struct {
	Domain            string     `gorm:"column:domain;primaryKey" json:"domain"`
	IsActive          bool       `gorm:"column:is_active" json:"is_active"`
	Frequency         string     `gorm:"column:frequency" json:"frequency"`
	RunTime           string     `gorm:"column:run_time" json:"run_time"`
	DayOfWeek         *string    `gorm:"column:day_of_week" json:"day_of_week"`
	DayOfMonth        *int       `gorm:"column:day_of_month" json:"day_of_month"`
	Timezone          string     `gorm:"column:timezone" json:"timezone"`
	SelectedModel     string     `gorm:"column:selected_model" json:"selected_model"`
	LastRunAt         *time.Time `gorm:"column:last_run_at" json:"last_run_at"`
	LastRunStatus     *string    `gorm:"column:last_run_status" json:"last_run_status"`
	LastRunMessage    *string    `gorm:"column:last_run_message" json:"last_run_message"`
	LastTrainingRunID *string    `gorm:"column:last_training_run_id" json:"last_training_run_id,omitempty"`
	NextRunAt         *time.Time `gorm:"column:next_run_at" json:"next_run_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at" json:"updated_at"`
	UpdatedBy         *string    `gorm:"column:updated_by" json:"updated_by"`
}

func (ForecastingSchedule) TableName() string { return "forecasting_schedules" }

// UpsertScheduleRequest is the PUT /forecasting/schedules/:domain body.
type UpsertScheduleRequest struct {
	IsActive      bool    `json:"is_active"`
	Frequency     string  `json:"frequency"`
	RunTime       string  `json:"run_time"`
	DayOfWeek     *string `json:"day_of_week"`
	DayOfMonth    *int    `json:"day_of_month"`
	Timezone      string  `json:"timezone"`
	SelectedModel string  `json:"selected_model"`
}

// ScheduleRunReport is the POST /forecasting/schedules/:domain/runs body
// (sent by the forecasting script).
type ScheduleRunReport struct {
	Status        string  `json:"status"` // RUNNING | SUCCEEDED | FAILED
	StartedAt     string  `json:"started_at"`
	FinishedAt    *string `json:"finished_at"`
	Message       *string `json:"message"`
	TrainingRunID *string `json:"training_run_id"`
}
