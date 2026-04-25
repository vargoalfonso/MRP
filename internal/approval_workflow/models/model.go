package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
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
type ApprovalLevel struct {
	Level      int    `json:"level"`
	Role       string `json:"role"`        // role required for this level; empty when skipped
	Status     string `json:"status"`      // pending | approved | rejected | skipped
	ApprovedBy string `json:"approved_by"` // user uuid; empty until actioned
	ApprovedAt string `json:"approved_at"` // RFC3339 timestamp; empty until actioned
	Note       string `json:"note"`        // optional note; empty until actioned
}

// ApprovalProgress is serialized as JSONB in approval_instances.approval_progress.
type ApprovalProgress struct {
	Levels []ApprovalLevel `json:"levels"`
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

type Claims struct {
	jwt.RegisteredClaims
	// UserID mirrors RegisteredClaims.Subject but is kept here for clarity.
	UserID string   `json:"uid"`
	Roles  []string `json:"roles"`
}

type ApprovalManagerSummary struct {
	Type     string `json:"type"`
	Pending  int64  `json:"pending"`
	Approved int64  `json:"approved"`
	Rejected int64  `json:"rejected"`
	Total    int64  `json:"total"`
}

type ApprovalManagerPagination struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"total_pages"`
}

type ApprovalManagerItem struct {
	InstanceID        int64  `json:"instance_id"`
	Module            string `json:"module"`
	ModuleLabel       string `json:"module_label"`
	ReferenceTable    string `json:"reference_table"`
	ReferenceID       int64  `json:"reference_id"`
	DocumentID        string `json:"document_id"`
	DocumentUUID      string `json:"document_uuid,omitempty"`
	ItemName          string `json:"item_name"`
	ItemCode          string `json:"item_code"`
	SubmittedBy       string `json:"submitted_by"`
	SubmittedByName   string `json:"submitted_by_name"`
	SubmittedAt       string `json:"submitted_at,omitempty"`
	Status            string `json:"status"`
	CurrentLevel      int    `json:"current_level"`
	MaxLevel          int    `json:"max_level"`
	CurrentLevelRole  string `json:"current_level_role"`
	IsFinalLevel      bool   `json:"is_final_level"`
	CanView           bool   `json:"can_view"`
	CanApprove        bool   `json:"can_approve"`
	CanReject         bool   `json:"can_reject"`
	IsMyTurn          bool   `json:"is_my_turn"`
	ViewMode          string `json:"view_mode"`
	DetailURL         string `json:"detail_url"`
	ApprovalURL       string `json:"approval_url"`
}

type ApprovalManagerListResponse struct {
	Items      []ApprovalManagerItem     `json:"items"`
	Pagination ApprovalManagerPagination `json:"pagination"`
}

type ApprovalManagerDetail struct {
	InstanceID       int64            `json:"instance_id"`
	Module           string           `json:"module"`
	ReferenceTable   string           `json:"reference_table"`
	ReferenceID      int64            `json:"reference_id"`
	Workflow         ApprovalWorkflow `json:"workflow"`
	CurrentLevel     int              `json:"current_level"`
	MaxLevel         int              `json:"max_level"`
	Status           string           `json:"status"`
	SubmittedBy      string           `json:"submitted_by"`
	ApprovalProgress ApprovalProgress `json:"approval_progress"`
	Document         ApprovalDocument `json:"document"`
	CanView          bool             `json:"can_view"`
	CanApprove       bool             `json:"can_approve"`
	CanReject        bool             `json:"can_reject"`
	IsMyTurn         bool             `json:"is_my_turn"`
	ViewMode         string           `json:"view_mode"`
}

type ApprovalDocument struct {
	ID        int64  `json:"id"`
	UUID      string `json:"uuid,omitempty"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	DetailURL string `json:"detail_url"`
}
