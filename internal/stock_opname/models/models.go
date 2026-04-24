package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

const (
	InventoryTypeFG  = "FG"
	InventoryTypeRM  = "RM"
	InventoryTypeIDR = "IDR"
	InventoryTypeWIP = "WIP"

	MethodManual = "manual"
	MethodBulk   = "bulk"

	SessionStatusDraft             = "draft"
	SessionStatusInProgress        = "in_progress"
	SessionStatusPendingApproval   = "pending_approval"
	SessionStatusApproved          = "approved"
	SessionStatusRejected          = "rejected"
	SessionStatusPartiallyApproved = "partially_approved"

	EntryStatusPending  = "pending"
	EntryStatusApproved = "approved"
	EntryStatusRejected = "rejected"

	ApprovalActionApprove = "approve"
	ApprovalActionReject  = "reject"

	AuditEntitySession = "session"
	AuditEntityEntry   = "entry"

	AuditActionCreate         = "create"
	AuditActionUpdate         = "update"
	AuditActionDelete         = "delete"
	AuditActionAddEntry       = "add_entry"
	AuditActionBulkAddEntries = "bulk_add_entries"
	AuditActionUpdateEntry    = "update_entry"
	AuditActionDeleteEntry    = "delete_entry"
	AuditActionSubmit         = "submit"
	AuditActionApproveSession = "approve_session"
	AuditActionApproveEntry   = "approve_entry"
	AuditActionRejectSession  = "reject_session"
	AuditActionRejectEntry    = "reject_entry"
)

var ValidInventoryTypes = map[string]struct{}{
	InventoryTypeFG:  {},
	InventoryTypeRM:  {},
	InventoryTypeIDR: {},
	InventoryTypeWIP: {},
}

type StockOpnameSession struct {
	ID                int64      `gorm:"primaryKey;autoIncrement"`
	UUID              uuid.UUID  `gorm:"type:uuid;uniqueIndex;not null"`
	SessionNumber     string     `gorm:"size:64;uniqueIndex;not null"`
	InventoryType     string     `gorm:"size:16;index;not null"`
	Method            string     `gorm:"size:16;not null"`
	PeriodMonth       int        `gorm:"not null"`
	PeriodYear        int        `gorm:"not null"`
	WarehouseLocation *string    `gorm:"size:255"`
	ScheduleDate      *time.Time `gorm:"type:date"`
	CountedDate       *time.Time `gorm:"type:date"`
	Remarks           *string    `gorm:"type:text"`

	TotalEntries     int     `gorm:"not null;default:0"`
	TotalVarianceQty float64 `gorm:"type:numeric(15,4);not null;default:0"`
	Status           string  `gorm:"size:32;index;not null"`

	SubmittedBy     *string `gorm:"size:255"`
	SubmittedAt     *time.Time
	Approver        *string `gorm:"size:255"`
	ApprovedBy      *string `gorm:"size:255"`
	ApprovedAt      *time.Time
	ApprovalRemarks *string    `gorm:"type:text"`
	CreatedBy       *string    `gorm:"size:255"`
	CreatedAt       time.Time  `gorm:"not null;default:now()"`
	UpdatedBy       *string    `gorm:"size:255"`
	UpdatedAt       time.Time  `gorm:"not null;default:now()"`
	DeletedAt       *time.Time `gorm:"index"`

	Entries []StockOpnameEntry `gorm:"foreignKey:SessionID"`
}

func (StockOpnameSession) TableName() string { return "stock_opname_sessions" }

type StockOpnameEntry struct {
	ID                int64     `gorm:"primaryKey;autoIncrement"`
	UUID              uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"`
	SessionID         int64     `gorm:"index;not null"`
	UniqCode          string    `gorm:"size:64;index;not null"`
	EntityID          *int64    `gorm:"index"`
	PartNumber        *string   `gorm:"size:128"`
	PartName          *string   `gorm:"size:255"`
	UOM               *string   `gorm:"size:32"`
	SystemQtySnapshot float64   `gorm:"type:numeric(15,4);not null"`
	CountedQty        float64   `gorm:"type:numeric(15,4);not null"`
	VarianceQty       float64   `gorm:"->;type:numeric(15,4)"`
	VariancePct       *float64  `gorm:"type:numeric(15,4)"`
	WeightKg          *float64  `gorm:"type:numeric(15,4)"`
	CyclePengiriman   *string   `gorm:"size:64"`
	UserCounter       *string   `gorm:"size:255"`
	Remarks           *string   `gorm:"type:text"`
	Status            string    `gorm:"size:16;index;not null"`
	ApprovedBy        *string   `gorm:"size:255"`
	ApprovedAt        *time.Time
	RejectReason      *string   `gorm:"type:text"`
	CreatedBy         *string   `gorm:"size:255"`
	CreatedAt         time.Time `gorm:"not null;default:now()"`
	UpdatedBy         *string   `gorm:"size:255"`
	UpdatedAt         time.Time `gorm:"not null;default:now()"`
}

func (StockOpnameEntry) TableName() string { return "stock_opname_entries" }

type StockOpnameAuditLog struct {
	ID            int64          `gorm:"primaryKey;autoIncrement"`
	UUID          uuid.UUID      `gorm:"type:uuid;uniqueIndex;not null"`
	SessionID     int64          `gorm:"index;not null"`
	EntryID       *int64         `gorm:"index"`
	InventoryType string         `gorm:"size:16;index;not null"`
	Action        string         `gorm:"size:64;index;not null"`
	EntityType    string         `gorm:"size:32;not null"`
	Actor         string         `gorm:"size:255;not null"`
	Remarks       *string        `gorm:"type:text"`
	Metadata      datatypes.JSON `gorm:"type:jsonb"`
	CreatedAt     time.Time      `gorm:"not null;default:now();index"`
}

func (StockOpnameAuditLog) TableName() string { return "stock_opname_audit_logs" }
