// Package models defines domain structs for the Bill of Material module.
package models

import (
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// DB Tables — matched 1:1 with migration 0003
// ---------------------------------------------------------------------------

type UomParameter struct {
	ID        int64 `gorm:"primaryKey;autoIncrement"`
	Code      string
	Name      string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (UomParameter) TableName() string { return "uom_parameters" }

type ProcessParameter struct {
	ID          int64 `gorm:"primaryKey;autoIncrement"`
	ProcessCode string
	ProcessName string
	Sequence    int
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (ProcessParameter) TableName() string { return "process_parameters" }

type MasterMachine struct {
	ID             int64 `gorm:"primaryKey;autoIncrement"`
	MachineNumber  string
	MachineName    string
	ProductionLine string
	ProcessID      *int64
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (MasterMachine) TableName() string { return "master_machines" }

// Supplier maps to the existing suppliers table (uuid PK, managed outside this module).
type Supplier struct {
	ID                   uuid.UUID `gorm:"type:uuid;primaryKey"`
	SupplierCode         string
	SupplierName         string
	TaxIdNpwp            *string
	BankAccountNumber    *string
	PaymentTerms         *string
	DeliveryLeadTimeDays *int
	Status               string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func (Supplier) TableName() string { return "suppliers" }

// Item — every UNIQ code (parent or child)
type Item struct {
	ID              int64     `gorm:"primaryKey;autoIncrement"`
	UUID            uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"`
	UniqCode        string    `gorm:"size:64;uniqueIndex;not null"`
	PartNumber      *string   `gorm:"size:128"`
	PartName        string    `gorm:"size:255;not null"`
	UomID           int64     `gorm:"not null"`
	CurrentRevision *string   `gorm:"size:32"`
	Status          string    `gorm:"size:20;default:Active"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time `gorm:"index"`
}

func (Item) TableName() string { return "items" }

type ItemRevision struct {
	ID         int64     `gorm:"primaryKey;autoIncrement"`
	UUID       uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"`
	ItemID     int64     `gorm:"not null;index"`
	Revision   string    `gorm:"size:32;not null"`
	Status     string    `gorm:"size:20;default:Draft"`
	ChangeNote *string   `gorm:"type:text"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (ItemRevision) TableName() string { return "item_revisions" }

type ItemMaterialSpec struct {
	ID             int64      `gorm:"primaryKey;autoIncrement"`
	ItemRevisionID int64      `gorm:"uniqueIndex;not null"`
	MaterialGrade  *string    `gorm:"size:64"`
	Form           *string    `gorm:"size:32"`
	WidthMm        *float64   `gorm:"type:numeric(18,4)"`
	DiameterMm     *float64   `gorm:"type:numeric(18,4)"`
	ThicknessMm    *float64   `gorm:"type:numeric(18,4)"`
	LengthMm       *float64   `gorm:"type:numeric(18,4)"`
	WeightKg       *float64   `gorm:"type:numeric(18,6)"`
	SupplierID     *uuid.UUID `gorm:"type:uuid"`
	CycleTimeSec   *float64   `gorm:"type:numeric(18,4)"`
	SetupTimeMin   *float64   `gorm:"type:numeric(18,4)"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (ItemMaterialSpec) TableName() string { return "item_material_specs" }

type ItemAsset struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	UUID      uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"`
	ItemID    int64     `gorm:"not null;index"`
	AssetType string    `gorm:"size:32;not null"`
	FileURL   string    `gorm:"type:text;not null"`
	Status    string    `gorm:"size:20;default:Active"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (ItemAsset) TableName() string { return "item_assets" }

type RoutingHeader struct {
	ID             int64     `gorm:"primaryKey;autoIncrement"`
	UUID           uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"`
	ItemID         int64     `gorm:"not null;index"`
	ItemRevisionID *int64    `gorm:"index"`
	Version        int       `gorm:"default:1"`
	Status         string    `gorm:"size:20;default:Draft"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (RoutingHeader) TableName() string { return "routing_headers" }

type RoutingOperation struct {
	ID              int64 `gorm:"primaryKey;autoIncrement"`
	RoutingHeaderID int64 `gorm:"not null;index"`
	OpSeq           int   `gorm:"not null"`
	ProcessID       int64 `gorm:"not null"`
	MachineID       *int64
	CycleTimeSec    *float64 `gorm:"type:numeric(18,4)"`
	SetupTimeMin    *float64 `gorm:"type:numeric(18,4)"`
	MachineStroke   *string  `gorm:"size:100"`
	Notes           *string  `gorm:"type:text"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (RoutingOperation) TableName() string { return "routing_operations" }

type RoutingOperationTooling struct {
	ID                 int64   `gorm:"primaryKey;autoIncrement"`
	RoutingOperationID int64   `gorm:"not null;index"`
	ToolingType        string  `gorm:"size:20;not null"`
	ToolingCode        *string `gorm:"size:100"`
	ToolingName        string  `gorm:"size:255;not null"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (RoutingOperationTooling) TableName() string { return "routing_operation_toolings" }

type BomItem struct {
	ID          int64     `gorm:"primaryKey;autoIncrement"`
	UUID        uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"`
	ItemID      int64     `gorm:"not null;index"`
	Version     int       `gorm:"default:1"`
	Status      string    `gorm:"size:20;default:Draft"`
	Description *string   `gorm:"type:text"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (BomItem) TableName() string { return "bom_item" }

type BomLine struct {
	ID           int64   `gorm:"primaryKey;autoIncrement"`
	BomItemID    int64   `gorm:"not null;index"`
	ParentItemID int64   `gorm:"not null;index"`
	ChildItemID  int64   `gorm:"not null;index"`
	Level        int16   `gorm:"default:1"`
	QtyPerUniq   float64 `gorm:"type:numeric(18,6);default:1"`
	UomID        *int64
	ScrapFactor  float64 `gorm:"type:numeric(9,6);default:0"`
	IsPhantom    bool    `gorm:"default:false"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (BomLine) TableName() string { return "bom_lines" }
