// Package approval provides a reusable helper for inserting rows into
// approval_instances.  Any module that needs multi-level approval
// (BOM, PO, DN, etc.) calls CreateInstance here instead of duplicating the logic.
//
// The approve/reject state machine is intentionally NOT in this package —
// each module has its own approve and reject endpoints that the frontend calls
// directly. This package only handles the creation side.
//
// Usage:
//
//	instance, err := approval.CreateInstance(ctx, db, approval.CreateInstanceParams{
//	    ActionName:     "bom",
//	    ReferenceTable: "bom_item",
//	    ReferenceID:    bom.ID,
//	    SubmittedBy:    userID,
//	})
package approval

import (
	"context"
	"fmt"

	awmodels "github.com/ganasa18/go-template/internal/approval_workflow/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// CreateInstance
// ---------------------------------------------------------------------------

// CreateInstanceParams contains all inputs needed to open a new approval process.
type CreateInstanceParams struct {
	// ActionName is the module identifier used to look up the workflow master.
	// Must match approval_workflows.action_name (e.g. "bom", "po", "dn").
	ActionName string

	// ReferenceTable is the table that owns the document (e.g. "bom_item", "purchase_orders").
	ReferenceTable string

	// ReferenceID is the PK of the document row.
	ReferenceID int64

	// SubmittedBy is the user UUID or username who triggered the submission.
	SubmittedBy string

	// MinLevels is the minimum number of approval levels required.
	// Defaults to 2 if zero.
	MinLevels int
}

// CreateInstance looks up the active workflow for ActionName, validates it,
// then inserts one row into approval_instances and returns it.
// Returns apperror.BadRequest if no workflow is configured or levels < MinLevels.
func CreateInstance(ctx context.Context, db *gorm.DB, p CreateInstanceParams) (*awmodels.ApprovalInstance, error) {
	minLevels := p.MinLevels
	if minLevels <= 0 {
		minLevels = 2
	}

	wf, err := getWorkflowByActionName(ctx, db, p.ActionName)
	if err != nil {
		return nil, err
	}
	if wf == nil {
		return nil, apperror.BadRequest(fmt.Sprintf(
			"no active approval workflow configured for action '%s'", p.ActionName,
		))
	}
	maxLevel := MaxLevel(wf)
	if maxLevel < minLevels {
		return nil, apperror.BadRequest(fmt.Sprintf(
			"approval workflow '%s' must have at least %d levels configured", p.ActionName, minLevels,
		))
	}

	instance := &awmodels.ApprovalInstance{
		ActionName:         p.ActionName,
		ReferenceTable:     p.ReferenceTable,
		ReferenceID:        p.ReferenceID,
		ApprovalWorkflowID: wf.ID,
		CurrentLevel:       1,
		MaxLevel:           maxLevel,
		Status:             "pending",
		SubmittedBy:        p.SubmittedBy,
		ApprovalProgress:   BuildProgress(wf, maxLevel),
	}
	if err := db.WithContext(ctx).Create(instance).Error; err != nil {
		return nil, fmt.Errorf("approval.CreateInstance: %w", err)
	}
	return instance, nil
}

// ---------------------------------------------------------------------------
// Pure helpers — exported so modules can use them without going through DB
// ---------------------------------------------------------------------------

// BuildProgress builds the initial JSONB ApprovalProgress from a workflow master.
// Levels 1..maxLevel are "pending"; the rest are "skipped".
func BuildProgress(wf *awmodels.ApprovalWorkflow, maxLevel int) awmodels.ApprovalProgress {
	roles := []string{wf.Level1Role, wf.Level2Role, wf.Level3Role, wf.Level4Role}
	levels := make([]awmodels.ApprovalLevel, 4)
	for i := 0; i < 4; i++ {
		lvl := i + 1
		status := "skipped"
		if lvl <= maxLevel {
			status = "pending"
		}
		levels[i] = awmodels.ApprovalLevel{
			Level:  lvl,
			Role:   roles[i],
			Status: status,
		}
	}
	return awmodels.ApprovalProgress{Levels: levels}
}

// LevelRole returns the required role for a given level number (1-4).
func LevelRole(wf *awmodels.ApprovalWorkflow, level int16) string {
	if wf == nil {
		return ""
	}
	switch level {
	case 1:
		return wf.Level1Role
	case 2:
		return wf.Level2Role
	case 3:
		return wf.Level3Role
	case 4:
		return wf.Level4Role
	}
	return ""
}

// MaxLevel returns the highest configured level (i.e. highest non-empty role) in a workflow.
func MaxLevel(wf *awmodels.ApprovalWorkflow) int {
	if wf == nil {
		return 1
	}
	max := 1
	if wf.Level2Role != "" {
		max = 2
	}
	if wf.Level3Role != "" {
		max = 3
	}
	if wf.Level4Role != "" {
		max = 4
	}
	return max
}

// HasRole reports whether any element of userRoles matches required.
func HasRole(userRoles []string, required string) bool {
	for _, r := range userRoles {
		if r == required {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Internal DB queries
// ---------------------------------------------------------------------------

func getWorkflowByActionName(ctx context.Context, db *gorm.DB, actionName string) (*awmodels.ApprovalWorkflow, error) {
	var wf awmodels.ApprovalWorkflow
	err := db.WithContext(ctx).
		Where("action_name = ? AND status = 'active'", actionName).
		First(&wf).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("approval: lookup workflow '%s': %w", actionName, err)
	}
	return &wf, nil
}
