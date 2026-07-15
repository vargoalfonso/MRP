package models

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	awmodels "github.com/ganasa18/go-template/internal/approval_workflow/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	PRLStatusPending  = "pending"
	PRLStatusApproved = "approved"
	PRLStatusRejected = "rejected"
)

var AllowedPRLStatuses = []string{
	PRLStatusPending,
	PRLStatusApproved,
	PRLStatusRejected,
}

const (
	PRLTypeReguler    = "reguler"
	PRLTypeAdditional = "additional"
)

var AllowedPRLTypes = []string{
	PRLTypeReguler,
	PRLTypeAdditional,
}

type UniqBillOfMaterial struct {
	ID           int64          `gorm:"primaryKey;autoIncrement" json:"-"`
	UUID         string         `gorm:"uniqueIndex;not null" json:"id"`
	UniqCode     string         `gorm:"uniqueIndex;not null" json:"uniq_code"`
	ProductModel string         `gorm:"not null" json:"product_model"`
	PartName     string         `gorm:"not null" json:"part_name"`
	PartNumber   string         `gorm:"not null" json:"part_number"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

type ItemLookup struct {
	UniqCode     string  `gorm:"column:uniq_code"`
	ProductModel *string `gorm:"column:model"`
	PartName     string  `gorm:"column:part_name"`
	PartNumber   *string `gorm:"column:part_number"`
}

type PRL struct {
	ID             int64          `gorm:"primaryKey;autoIncrement" json:"row_id"`
	UUID           string         `gorm:"uniqueIndex;not null" json:"id"`
	PRLID          string         `gorm:"not null" json:"prl_id"`
	PRLGroupUUID   *string        `gorm:"index" json:"prl_group_uuid,omitempty"`
	CustomerUUID   string         `gorm:"index;not null" json:"customer_uuid"`
	CustomerCode   string         `gorm:"not null" json:"customer_code"`
	CustomerName   string         `gorm:"not null" json:"customer_name"`
	UniqBOMUUID    string         `gorm:"index;not null" json:"uniq_bom_uuid"`
	UniqCode       string         `gorm:"not null" json:"uniq_code"`
	ProductModel   string         `gorm:"not null" json:"product_model"`
	PartName       string         `gorm:"not null" json:"part_name"`
	PartNumber     string         `gorm:"not null" json:"part_number"`
	ForecastPeriod string         `gorm:"type:text;not null" json:"forecast_period"`
	Quantity       int64          `gorm:"not null" json:"quantity"`
	Status         string         `gorm:"not null;default:'pending'" json:"status"`
	PRLType        string         `gorm:"type:varchar(20);not null;default:'reguler'" json:"prl_type"`
	Remarks        *string        `gorm:"type:text" json:"remarks,omitempty"`
	ApprovedAt     *time.Time     `gorm:"default:null" json:"approved_at,omitempty"`
	RejectedAt     *time.Time     `gorm:"default:null" json:"rejected_at,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	Note           string         `gorm:"type:text" json:"note,omitempty"`
	ChildJSONB     datatypes.JSON `gorm:"column:child_jsonb;type:jsonb;not null;default:'[]'::jsonb" json:"child_jsonb,omitempty"`
}

type PRLGroup struct {
	ID        int64          `gorm:"primaryKey;autoIncrement" json:"-"`
	UUID      string         `gorm:"uniqueIndex;not null" json:"id"`
	PRLID     string         `gorm:"uniqueIndex;not null" json:"prl_id"`
	Remarks   *string        `gorm:"type:text" json:"remarks,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type CreateUniqBOMRequest struct {
	UniqCode     string `json:"uniq_code" validate:"required,max=100"`
	ProductModel string `json:"product_model" ,max=255"`
	PartName     string `json:"part_name" validate:"required,max=255"`
	PartNumber   string `json:"part_number" validate:"required,max=150"`
}

type UpdateUniqBOMRequest struct {
	UniqCode     string `json:"uniq_code" validate:"required,max=100"`
	ProductModel string `json:"product_model" ,max=255"`
	PartName     string `json:"part_name" validate:"required,max=255"`
	PartNumber   string `json:"part_number" validate:"required,max=150"`
}

type ListUniqBOMQuery struct {
	Search string `form:"search"`
	Page   int    `form:"page"`
	Limit  int    `form:"limit"`
}

// CustomerUUIDInput supports either:
//   - JSON number: 123            (interpreted as customers.id / row id)
//   - JSON string: "<uuid>"      (interpreted as customers.uuid)
//   - JSON string: "123"         (also treated as row id for convenience)
type CustomerUUIDInput struct {
	UUID  string
	RowID int64
	IsInt bool
}

func (c *CustomerUUIDInput) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		*c = CustomerUUIDInput{}
		return nil
	}

	// Prefer string decoding first.
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		s = strings.TrimSpace(s)
		if s == "" {
			*c = CustomerUUIDInput{}
			return nil
		}
		if id, convErr := strconv.ParseInt(s, 10, 64); convErr == nil {
			c.UUID = ""
			c.RowID = id
			c.IsInt = true
			return nil
		}
		c.UUID = s
		c.RowID = 0
		c.IsInt = false
		return nil
	}

	// Fall back to number decoding.
	var n json.Number
	if err := json.Unmarshal(data, &n); err == nil {
		id, convErr := n.Int64()
		if convErr != nil {
			return fmt.Errorf("customer_uuid must be an integer")
		}
		c.UUID = ""
		c.RowID = id
		c.IsInt = true
		return nil
	}

	return fmt.Errorf("customer_uuid must be a string UUID or integer")
}

type FlexibleStringList []string

func (l *FlexibleStringList) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		*l = nil
		return nil
	}

	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		*l = normalizeStringList(arr)
		return nil
	}

	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*l = normalizeStringList(splitCSVLikeString(s))
		return nil
	}

	return fmt.Errorf("must be a string or array of strings")
}

func normalizeStringList(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func splitCSVLikeString(value string) []string {
	if !strings.Contains(value, ",") {
		return []string{value}
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		out = append(out, part)
	}
	return out
}

func (r CreatePRLRequest) AllUniqCodes() []string {
	values := make([]string, 0, len(r.UniqCode)+len(r.UniqCodes))
	values = append(values, r.UniqCode...)
	values = append(values, r.UniqCodes...)
	return normalizeStringList(values)
}

func PickFlexibleString(values FlexibleStringList, index int) string {
	if len(values) == 0 {
		return ""
	}
	if index >= 0 && index < len(values) {
		return strings.TrimSpace(values[index])
	}
	if index >= len(values) {
		return strings.TrimSpace(values[len(values)-1])
	}
	return ""
}

type CreatePRLRequest struct {
	CustomerUUID   CustomerUUIDInput  `json:"customer_uuid" validate:"-"`
	CustomerCode   string             `json:"customer_code" validate:"omitempty,max=32"`
	UniqCode       FlexibleStringList `json:"uniq_code" validate:"omitempty,dive,required,max=100"`
	UniqCodes      FlexibleStringList `json:"uniq_codes" validate:"omitempty,dive,required,max=100"`
	ProductModel   FlexibleStringList `json:"product_model" validate:"-"`
	PartName       FlexibleStringList `json:"part_name" validate:"-"`
	PartNumber     FlexibleStringList `json:"part_number" validate:"-"`
	ForecastPeriod string             `json:"forecast_period" validate:"required"`
	Quantity       int64              `json:"quantity" validate:"required,gte=1"`
	PRLType        string             `json:"prl_type" validate:"omitempty,oneof=reguler additional"`
	Remarks        string             `json:"remarks" validate:"omitempty"`
}

type CreatePRLEntryRequest struct {
	CustomerUUID   CustomerUUIDInput `json:"customer_uuid" validate:"-"`
	CustomerCode   string            `json:"customer_code" validate:"omitempty,max=32"`
	UniqCode       string            `json:"uniq_code" validate:"required,max=100"`
	ProductModel   string            `json:"product_model" validate:"omitempty,max=255"`
	PartName       string            `json:"part_name" validate:"omitempty,max=255"`
	PartNumber     string            `json:"part_number" validate:"omitempty,max=150"`
	ForecastPeriod string            `json:"forecast_period" validate:"required"`
	Quantity       int64             `json:"quantity" validate:"required,gte=1"`
	PRLType        string            `json:"prl_type" validate:"omitempty,oneof=reguler additional"`
	Remarks        string            `json:"remarks" validate:"omitempty"`
}

type BulkCreatePRLRequest struct {
	Entries []CreatePRLEntryRequest `json:"entries"`
}

type UpdatePRLRequest struct {
	ForecastPeriod string `json:"forecast_period" validate:"required"`
	Quantity       int64  `json:"quantity" validate:"required,gte=1"`
}

type BulkStatusActionRequest struct {
	IDs  []string `json:"ids"`
	Note string   `json:"note"`
}

type ListPRLQuery struct {
	Search         string `form:"search"`
	Status         string `form:"status"`
	ForecastPeriod string `form:"forecast_period"`
	CustomerUUID   string `form:"customer_uuid"`
	UniqCode       string `form:"uniq_code"`
	Page           int    `form:"page"`
	Limit          int    `form:"limit"`
}

type ListPRLHistoryQuery struct {
	Search         string `form:"search"`
	ForecastPeriod string `form:"forecast_period"`
	UniqCode       string `form:"uniq_code"`
	Page           int    `form:"page"`
	Limit          int    `form:"limit"`
}

type PRLHistoryDetailQuery struct {
	UniqCode       string `form:"uniq_code" validate:"required,max=100"`
	ForecastPeriod string `form:"forecast_period" validate:"required,max=100"`
	Limit          int    `form:"limit"`
}

type PaginationMeta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

type UniqBOMListResult struct {
	Items      []UniqBillOfMaterial `json:"items"`
	Pagination PaginationMeta       `json:"pagination"`
}

type PRLListResult struct {
	Items      []PRL          `json:"items"`
	Pagination PaginationMeta `json:"pagination"`
}

type PRLHistoryListItem struct {
	ForecastPeriod string    `json:"forecast_period"`
	UniqCode       string    `json:"uniq_code"`
	PRLQuantity    float64   `json:"prl_quantity"`
	DeliveryQty    float64   `json:"delivery_qty"`
	LastUpdated    time.Time `json:"last_updated"`
}

type PRLHistoryListResponse struct {
	Items      []PRLHistoryListItem `json:"items"`
	Pagination PaginationMeta       `json:"pagination"`
}

type PRLMachinePatternListItem struct {
	ForecastPeriod   string  `json:"forecast_period"`
	UniqCode         string  `json:"uniq_code"`
	MachinePattern   string  `json:"machine_pattern"`
	ProductionOutput float64 `json:"production_output"`
}

type PRLMachinePatternListResponse struct {
	Items      []PRLMachinePatternListItem `json:"items"`
	Pagination PaginationMeta              `json:"pagination"`
}

type PRLHistoryLogItem struct {
	Activity    string    `json:"activity"`
	Description string    `json:"description"`
	Actor       string    `json:"actor"`
	EventTime   time.Time `json:"event_time"`
	Source      string    `json:"source"`
	OldValue    *float64  `json:"old_value,omitempty"`
	NewValue    *float64  `json:"new_value,omitempty"`
}

type PRLHistoryDetailSummary struct {
	UniqCode       string    `json:"uniq_code"`
	ForecastPeriod string    `json:"forecast_period"`
	TotalLogs      int64     `json:"total_logs"`
	PRLQuantity    float64   `json:"prl_quantity"`
	DeliveryQty    float64   `json:"delivery_qty"`
	LastUpdated    time.Time `json:"last_updated"`
	MachinePattern string    `json:"machine_pattern"`
}

type PRLHistoryDetailResponse struct {
	Summary  PRLHistoryDetailSummary `json:"summary"`
	Timeline []PRLHistoryLogItem     `json:"timeline"`
}

type PRLDetailResponse struct {
	PRL      PRL                 `json:"prl"`
	Approval *PRLApprovalSummary `json:"approval,omitempty"`
}

type ChildInfo struct {
	UniqCode      string `json:"uniq_code"`
	MaterialGrade string `json:"material_grade"`
}

type PRLApprovalSummary struct {
	InstanceID       int64                     `json:"instance_id"`
	WorkflowID       int64                     `json:"workflow_id"`
	WorkflowAction   string                    `json:"workflow_action"`
	CurrentLevel     int                       `json:"current_level"`
	MaxLevel         int                       `json:"max_level"`
	Status           string                    `json:"status"`
	SubmittedBy      string                    `json:"submitted_by"`
	ApprovalProgress awmodels.ApprovalProgress `json:"approval_progress"`
	LevelRoles       PRLApprovalWorkflowRoles  `json:"level_roles"`
}

type PRLApprovalWorkflowRoles struct {
	Level1 string `json:"level_1_role,omitempty"`
	Level2 string `json:"level_2_role,omitempty"`
	Level3 string `json:"level_3_role,omitempty"`
	Level4 string `json:"level_4_role,omitempty"`
}

type UniqBOMListFilters struct {
	Search string
	Page   int
	Limit  int
	Offset int
}

type PRLListFilters struct {
	Search         string
	Status         *string
	ForecastPeriod *string
	CustomerUUID   *string
	UniqCode       *string
	Page           int
	Limit          int
	Offset         int
}

type PRLHistoryFilters struct {
	Search         string
	ForecastPeriod string
	UniqCode       string
	Page           int
	Limit          int
	Offset         int
}

type CustomerLookup struct {
	ID           string `json:"id"`
	CustomerID   string `json:"customer_id"`
	CustomerName string `json:"customer_name"`
}

type ForecastPeriodOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type BulkCreatePRLResponse struct {
	CreatedCount int   `json:"created_count"`
	Items        []PRL `json:"items"`
}

type BulkStatusActionResponse struct {
	UpdatedCount int64                    `json:"updated_count"`
	Status       string                   `json:"status"`
	Results      []BulkStatusActionResult `json:"results,omitempty"`
}

type BulkStatusActionResult struct {
	ID           string `json:"id"`
	CurrentLevel int    `json:"current_level"`
	Status       string `json:"status"`
}

type ImportPRLResponse struct {
	ImportedCount int   `json:"imported_count"`
	Items         []PRL `json:"items"`
}

func NewPaginationMeta(page, limit int, total int64) PaginationMeta {
	totalPages := 0
	if limit > 0 && total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(limit)))
	}

	return PaginationMeta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}
}

func Trimmed(value string) string {
	return strings.TrimSpace(value)
}

type ApproveInstance struct {
	ID                 int64     `gorm:"column:id;primaryKey;autoIncrement"`
	ActionName         string    `gorm:"column:action_name"`
	ReferenceTable     string    `gorm:"column:reference_table"`
	ReferenceID        uint      `gorm:"column:reference_id"`
	ApprovalWorkflowID uint      `gorm:"column:approval_workflow_id"`
	CurrentLevel       int       `gorm:"column:current_level"`
	MaxLevel           int       `gorm:"column:max_level"`
	Status             string    `gorm:"column:status"`
	SubmittedBy        uint      `gorm:"column:submitted_by"`
	ApprovalProgress   string    `gorm:"column:approval_progress"`
	CreatedAt          time.Time `gorm:"column:created_at"`
	UpdatedAt          time.Time `gorm:"column:updated_at"`
}

func (ApproveInstance) TableName() string {
	// comment
	return "approval_instances"
}
