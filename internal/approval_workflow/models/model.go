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
