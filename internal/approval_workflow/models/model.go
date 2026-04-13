package models

import (
	"time"
)

type ApprovalWorkflow struct {
	ID         int64     `gorm:"primaryKey"`
	ActionName string    `gorm:"column:action_name"`
	Level1Role string    `gorm:"column:level_1_role"`
	Level2Role string    `gorm:"column:level_2_role"`
	Level3Role string    `gorm:"column:level_3_role"`
	Level4Role string    `gorm:"column:level_4_role"`
	Status     string    `gorm:"column:status"`
	CreatedBy  string    `gorm:"column:created_by"`
	CreatedAt  time.Time `gorm:"column:created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at"`
}

func (ApprovalWorkflow) TableName() string {
	return "approval_workflows"
}

// ---------------------------------------------------------------------------
// approval_instances — generic multi-level approval tracker
// ---------------------------------------------------------------------------

// ApprovalLevelProgress holds the per-level state stored inside approval_progress JSONB.
type ApprovalLevelProgress struct {
	Level      int    `json:"level"`
	Role       string `json:"role"`        // role required for this level; empty when skipped
	Status     string `json:"status"`      // pending | approved | rejected | skipped
	ApprovedBy string `json:"approved_by"` // user uuid; empty until actioned
	ApprovedAt string `json:"approved_at"` // RFC3339 timestamp; empty until actioned
	Note       string `json:"note"`        // optional note; empty until actioned
}

// ApprovalProgress is serialized as JSONB in approval_instances.approval_progress.
type ApprovalProgress struct {
	Levels []ApprovalLevelProgress `json:"levels"`
}

// ApprovalInstance tracks one approval process for any document.
// action_name identifies the module; reference_table + reference_id pinpoint the document.
type ApprovalInstance struct {
	ID                 int64            `gorm:"primaryKey;autoIncrement"`
	ActionName         string           `gorm:"column:action_name"`
	ReferenceTable     string           `gorm:"column:reference_table"`
	ReferenceID        int64            `gorm:"column:reference_id"`
	ApprovalWorkflowID int64            `gorm:"column:approval_workflow_id"`
	CurrentLevel       int              `gorm:"column:current_level;default:1"`
	MaxLevel           int              `gorm:"column:max_level"`
	Status             string           `gorm:"column:status;default:pending"` // pending | approved | rejected
	SubmittedBy        string           `gorm:"column:submitted_by"`
	ApprovalProgress   ApprovalProgress `gorm:"column:approval_progress;type:jsonb;serializer:json"`
	CreatedAt          time.Time        `gorm:"column:created_at"`
	UpdatedAt          time.Time        `gorm:"column:updated_at"`
}

func (ApprovalInstance) TableName() string { return "approval_instances" }
