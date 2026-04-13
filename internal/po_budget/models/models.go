// Package models defines domain structs for the PO Budget module.
package models

import "time"

// ---------------------------------------------------------------------------
// DB Tables
// ---------------------------------------------------------------------------

// POSplitSetting stores the PO1/PO2 percentage split per budget type.
// Default: PO1=60%, PO2=40% (kanban packing logic).
// One PO is always split into two stages — e.g. 60% at first delivery, 40% on completion.
type POSplitSetting struct {
	ID            int64     `gorm:"primaryKey;autoIncrement"`
	BudgetType    string    `gorm:"size:32;uniqueIndex;not null"`
	Po1Pct        float64   `gorm:"type:numeric(5,2);not null;default:60"`
	Po2Pct        float64   `gorm:"type:numeric(5,2);not null;default:40"`
	MinOrderQty   *int64    `gorm:""`        // optional; for PO line splitting rules
	MaxSplitLines *int64    `gorm:""`        // optional; for PO line splitting rules
	SplitRule     *string   `gorm:"size:64"` // optional; e.g. "By Supplier Capacity"
	Status        string    `gorm:"size:20;not null;default:Active"`
	Description   string    `gorm:"type:text"`
	CreatedAt     time.Time `gorm:"not null;default:now()"`
	UpdatedAt     time.Time `gorm:"not null;default:now()"`
}

func (POSplitSetting) TableName() string { return "po_split_settings" }

// POBudgetEntry is one row in the PO budget for a given Uniq / customer / period.
// Multiple rows with the same uniq_code + period are aggregated for the summary view.
type POBudgetEntry struct {
	ID int64 `gorm:"primaryKey;autoIncrement"`
	// PoBudgetRef is a DB-generated identifier: POB-{YYYY}-{TYPE}-{id}
	PoBudgetRef     string    `gorm:"size:32;->"`
	BudgetType      string    `gorm:"size:32;not null"` // raw_material | subcon | indirect
	CustomerID      *int64    `gorm:"index"`
	CustomerName    *string   `gorm:"size:255"`
	UniqCode        string    `gorm:"size:64;not null;index"`
	ProductModel    *string   `gorm:"size:255"`
	MaterialType    *string   `gorm:"size:64"` // Pipe | Steel Plate | Wire | Add On
	PartName        *string   `gorm:"size:255"`
	PartNumber      *string   `gorm:"size:128"`
	Quantity        float64   `gorm:"type:numeric(15,4);not null;default:0"`
	Uom             *string   `gorm:"size:32"`
	WeightKg        *float64  `gorm:"type:numeric(15,4)"`
	Description     *string   `gorm:"type:text"`
	SupplierID      *string   `gorm:"size:36"` // UUID as string
	SupplierName    *string   `gorm:"size:255"`
	Period          string    `gorm:"size:32;not null"` // e.g. "October 2025"
	PeriodDate      time.Time `gorm:"type:date;not null;index"`
	SalesPlan       float64   `gorm:"type:numeric(15,4);not null;default:0"`
	PurchaseRequest float64   `gorm:"type:numeric(15,4);not null;default:0"`
	Po1Pct          float64   `gorm:"type:numeric(5,2);not null;default:60"`
	Po2Pct          float64   `gorm:"type:numeric(5,2);not null;default:40"`
	Po1Qty          float64   `gorm:"type:numeric(15,4);->;-:migration"` // GENERATED ALWAYS
	Po2Qty          float64   `gorm:"type:numeric(15,4);->;-:migration"` // GENERATED ALWAYS
	TotalPO         float64   `gorm:"type:numeric(15,4);->;-:migration"` // GENERATED ALWAYS
	Prl             float64   `gorm:"type:numeric(15,4);not null;default:0"`
	Status          string    `gorm:"size:32;not null;default:Draft"`
	ApprovedBy      *string   `gorm:"size:255"`
	ApprovedAt      *time.Time

	// PRL linkage (only for bulk-from-PRL entries)
	// Legacy (migration 0010) linkage to prl_forecasts/prl_forecast_items.
	// Kept for backward compatibility but not used for the "data asli" PRL flow.
	// Marked read-only so inserts won't require these columns.
	PrlID     *int64 `gorm:"index;->"`
	PrlItemID *int64 `gorm:"->"`

	// Data-asli PRL linkage (table: prls)
	PrlRef   *string `gorm:"size:32;index"` // prls.prl_id
	PrlRowID *int64  `gorm:"index"`         // prls.id

	BudgetQty     *float64 `gorm:"type:numeric(15,4)"` // PRL item qty ceiling (snapshot)
	BudgetSubtype *string  `gorm:"size:32"`            // adhoc | regular | nil

	// Audit
	CreatedBy *string   `gorm:"size:255"`
	UpdatedBy *string   `gorm:"size:255"`
	CreatedAt time.Time `gorm:"not null;default:now()"`
	UpdatedAt time.Time `gorm:"not null;default:now()"`
}

func (POBudgetEntry) TableName() string { return "po_budget_entries" }

// POBudgetEntryLog records the audit/history trail for a PO budget entry.
type POBudgetEntryLog struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	EntryID   int64     `gorm:"not null;index"`
	Action    string    `gorm:"size:32;not null"` // Created|Submitted|Updated|Approved|Rejected
	Username  *string   `gorm:"size:255"`
	Notes     *string   `gorm:"type:text"`
	CreatedAt time.Time `gorm:"not null;default:now()"`
}

func (POBudgetEntryLog) TableName() string { return "po_budget_entry_logs" }

// ---------------------------------------------------------------------------
// PRL Forecast
// ---------------------------------------------------------------------------

// PrlForecast is the header of a Production Requirement List per customer+period.
type PrlForecast struct {
	ID           int64     `gorm:"primaryKey;autoIncrement"`
	PrlNumber    string    `gorm:"size:64;not null;uniqueIndex:prl_number_period_uq"`
	CustomerID   *int64    `gorm:"index"`
	CustomerName *string   `gorm:"size:255"`
	Period       string    `gorm:"size:32;not null"`
	PeriodDate   time.Time `gorm:"type:date;not null;index"`
	Status       string    `gorm:"size:32;not null;default:Active"`
	Notes        *string   `gorm:"type:text"`
	CreatedBy    *string   `gorm:"size:255"`
	CreatedAt    time.Time `gorm:"not null;default:now()"`
	UpdatedAt    time.Time `gorm:"not null;default:now()"`

	Items []PrlForecastItem `gorm:"foreignKey:PrlID"`
}

func (PrlForecast) TableName() string { return "prl_forecasts" }

// PrlForecastItem is one UNIQ row within a PRL.
// `Quantity` is the budget ceiling — sum of all supplier allocations for this
// item in po_budget_entries MUST NOT exceed this value.
type PrlForecastItem struct {
	ID                  int64     `gorm:"primaryKey;autoIncrement"`
	PrlID               int64     `gorm:"not null;index"`
	UniqCode            string    `gorm:"size:64;not null;index"`
	PartName            *string   `gorm:"size:255"`
	PartNumber          *string   `gorm:"size:128"`
	WeightKg            *float64  `gorm:"type:numeric(15,4)"`
	Quantity            float64   `gorm:"type:numeric(15,4);not null"` // budget ceiling
	ExistingRawMaterial *string   `gorm:"size:255"`
	Uom                 *string   `gorm:"size:32"`
	CreatedAt           time.Time `gorm:"not null;default:now()"`
	UpdatedAt           time.Time `gorm:"not null;default:now()"`
}

func (PrlForecastItem) TableName() string { return "prl_forecast_items" }

// ---------------------------------------------------------------------------
// PRL (data asli)
// ---------------------------------------------------------------------------

// PRLRow maps to the existing "prls" table (data asli).
// One row represents one UNIQ item within a PRL document (prl_id).
type PRLRow struct {
	ID             int64      `gorm:"primaryKey;autoIncrement"`
	UUID           string     `gorm:"type:uuid"`
	PrlID          string     `gorm:"size:32;index"` // varchar(32)
	CustomerUUID   *string    `gorm:"type:uuid"`
	CustomerCode   *string    `gorm:"size:32"`
	CustomerName   *string    `gorm:"size:255"`
	UniqBomUUID    *string    `gorm:"type:uuid"`
	UniqCode       *string    `gorm:"size:100"`
	ProductModel   *string    `gorm:"size:255"`
	PartName       *string    `gorm:"size:255"`
	PartNumber     *string    `gorm:"size:150"`
	ForecastPeriod *string    `gorm:"size:7"` // MM-YYYY
	Quantity       float64    `gorm:"type:numeric(15,4)"`
	Status         *string    `gorm:"size:20"`
	ApprovedAt     *time.Time `gorm:"type:timestamptz"`
	RejectedAt     *time.Time `gorm:"type:timestamptz"`
	CreatedAt      time.Time  `gorm:"type:timestamptz"`
	UpdatedAt      time.Time  `gorm:"type:timestamptz"`
	DeletedAt      *time.Time `gorm:"type:timestamp"`
}

func (PRLRow) TableName() string { return "prls" }
