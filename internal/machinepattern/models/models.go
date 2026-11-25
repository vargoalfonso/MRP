// Package models defines domain structs for the Machine Pattern module.
package models

import "time"

// ---------------------------------------------------------------------------
// DB Tables
// ---------------------------------------------------------------------------

// MachinePattern stores one pattern record per UNIQ-machine pair.
type MachinePattern struct {
	ID           int64   `gorm:"primaryKey;autoIncrement"`
	UniqCode     string  `gorm:"size:64;not null;index"`
	MachineID    int64   `gorm:"not null;index"`
	CycleTimeSec float64 `gorm:"type:numeric(18,4);not null"` // seconds — auto-filled from BOM
	PrlReference float64 `gorm:"type:numeric(18,4);not null"` // total PRL qty for this UNIQ
	PatternValue float64 `gorm:"type:numeric(18,4);not null"` // calculated or user-overridden
	WorkingDays  int     `gorm:"not null;default:25"`
	MovingType   string  `gorm:"size:20;not null;default:'Normal'"` // Fast Moving | Slow Moving | Normal
	MinOutput    float64 `gorm:"type:numeric(18,4);not null"`       // PRL/wd * CycleTimeSec * PatternValue
	Status       string  `gorm:"size:20;not null;default:'Active'"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (MachinePattern) TableName() string { return "machine_patterns" }

// MachinePatternParam holds global parameterised thresholds (singleton row, id=1).
type MachinePatternParam struct {
	ID                  int64   `gorm:"primaryKey;autoIncrement"`
	FastMovingThreshold float64 `gorm:"type:numeric(18,4);not null;default:1000"` // C for Fast Moving
	SlowMovingThreshold float64 `gorm:"type:numeric(18,4);not null;default:1000"` // C for Slow Moving
	PatternMinMinutes   float64 `gorm:"type:numeric(18,4);not null;default:48"`   // C for pattern calc (minutes)
	DefaultWorkingDays  int     `gorm:"not null;default:25"`
	UpdatedAt           time.Time
}

func (MachinePatternParam) TableName() string { return "machine_pattern_params" }

// ---------------------------------------------------------------------------
// Lightweight reference structs (cross-module reads, no FK)
// ---------------------------------------------------------------------------

type RefMachine struct {
	ID          int64
	MachineName string
}

// ---------------------------------------------------------------------------
// Request / Response DTOs
// ---------------------------------------------------------------------------

// CreateMachinePatternRequest — single create
type CreateMachinePatternRequest struct {
	UniqCode     string  `json:"uniq_code"     validate:"required"`
	MachineID    int64   `json:"machine_id"    validate:"required,min=1"`
	CycleTimeSec float64 `json:"cycle_time_sec"`          // canonical (seconds)
	CycleTime    float64 `json:"cycle_time"`              // alias accepted from frontend
	PrlReference float64 `json:"prl_reference" validate:"min=0"` // 0 is a valid value (no PRL yet)
	WorkingDays  int     `json:"working_days"  validate:"required,min=1"`
	PatternValue float64 `json:"pattern_value"`           // optional client hint (server recalculates)
	MovingType   string  `json:"moving_type"`             // optional client hint (server recalculates)
	MinOutput    float64 `json:"min_output" validate:"min=0"`    // 0 allowed; server recomputes
	Status       string  `json:"status"`
}

// ResolveCycleTimeSec returns the cycle time in seconds, accepting either the
// canonical "cycle_time_sec" field or the frontend "cycle_time" alias.
func (r CreateMachinePatternRequest) ResolveCycleTimeSec() float64 {
	if r.CycleTimeSec > 0 {
		return r.CycleTimeSec
	}
	return r.CycleTime
}

// UpdateMachinePatternRequest — partial update
type UpdateMachinePatternRequest struct {
	CycleTimeSec *float64 `json:"cycle_time_sec"`
	PrlReference *float64 `json:"prl_reference"`
	WorkingDays  *int     `json:"working_days"`
	MinOutput    *float64 `json:"min_output"`
	Status       *string  `json:"status"`
}

// BulkCreateRequest — CSV/JSON bulk payload
type BulkCreateRequest struct {
	Items []CreateMachinePatternRequest `json:"items" validate:"required,min=1,dive"`
}

// UpdateParamRequest — update global params
type UpdateParamRequest struct {
	FastMovingThreshold *float64 `json:"fast_moving_threshold"`
	SlowMovingThreshold *float64 `json:"slow_moving_threshold"`
	PatternMinMinutes   *float64 `json:"pattern_min_minutes"`
	DefaultWorkingDays  *int     `json:"default_working_days"`
}

// CalculateRequest — preview calculation (no DB write)
type CalculateRequest struct {
	UniqCode     string  `json:"uniq_code"     form:"uniq_code"`
	MachineID    int64   `json:"machine_id"    form:"machine_id"`
	CycleTimeSec float64 `json:"cycle_time_sec" form:"cycle_time_sec"`
	PrlReference float64 `json:"prl_reference"  form:"prl_reference"`
	WorkingDays  int     `json:"working_days"   form:"working_days"`
}

// CalculateResult — output of the calculation preview
type CalculateResult struct {
	DailyRequirement float64 `json:"daily_requirement"`
	MovingType       string  `json:"moving_type"`
	PatternValue     float64 `json:"pattern_value"`
	MinOutput        float64 `json:"min_output"`
}

// MachinePatternResponse — single record response (enriched)
type MachinePatternResponse struct {
	ID           int64   `json:"id"`
	UniqCode     string  `json:"uniq_code"`
	MachineName  string  `json:"machine_name"`
	MachineID    int64   `json:"machine_id"`
	CycleTimeSec float64 `json:"cycle_time_sec"`
	CycleTime    float64 `json:"cycle_time"`
	PrlReference float64 `json:"prl_reference"`
	PatternValue float64 `json:"pattern_value"`
	WorkingDays  int     `json:"working_days"`
	MovingType   string  `json:"moving_type"`
	MinOutput    float64 `json:"min_output"`
	Status       string  `json:"status"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

// ListMachinePatternResponse — paginated list
type ListMachinePatternResponse struct {
	Total int64                      `json:"total"`
	Page  int                        `json:"page"`
	Limit int                        `json:"limit"`
	Items []MachinePatternResponse   `json:"items"`
}

// ListQuery — filter/pagination input
type ListQuery struct {
	UniqCode   string
	MachineID  int64
	MovingType string
	Status     string
	Search     string
	Page       int
	Limit      int
}

// BulkCreateResponse — summary for bulk
type BulkCreateResponse struct {
	Created int                        `json:"created"`
	Failed  int                        `json:"failed"`
	Errors  []string                   `json:"errors,omitempty"`
	Items   []MachinePatternResponse   `json:"items"`
}

// SafetyStockOutput — derived output for safety stock
type SafetyStockOutput struct {
	UniqCode        string  `json:"uniq_code"`
	MachineName     string  `json:"machine_name"`
	DailyOutput     float64 `json:"daily_output"`    // min_output / cycle_time_sec (per day)
	PatternValue    float64 `json:"pattern_value"`
	MovingType      string  `json:"moving_type"`
	SafetyStockQty  float64 `json:"safety_stock_qty"` // daily_output * working_days
}

// SummaryResponse — aggregate counts for the machine pattern dashboard cards.
type SummaryResponse struct {
	TotalPattern int64   `json:"total_pattern"`
	FastMoving   int64   `json:"fast_moving"`
	SlowMoving   int64   `json:"slow_moving"`
	Normal       int64   `json:"normal"`
	AvgPattern   float64 `json:"avg_pattern"`
}
