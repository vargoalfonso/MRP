package models

import "time"

// ─── Delivery Schedule Customer ──────────────────────────────────────────────

type ScheduleCustomer struct {
	ID                        int64      `gorm:"primaryKey;autoIncrement"`
	UUID                      string     `gorm:"type:uuid;uniqueIndex;not null"`
	ScheduleNumber            string     `gorm:"size:64;uniqueIndex;not null"`
	CustomerOrderDocumentID   *int64     `gorm:"index"`
	CustomerOrderReference    *string    `gorm:"size:128"`
	CustomerID                int64      `gorm:"not null;index"`
	CustomerNameSnapshot      *string    `gorm:"size:255"`
	CustomerContactPerson     *string    `gorm:"size:255"`
	CustomerPhoneNumber       *string    `gorm:"size:64"`
	DeliveryAddress           *string    `gorm:"type:text"`
	ScheduleDate              time.Time  `gorm:"type:date;not null;index"`
	Priority                  string     `gorm:"size:16;not null;default:'normal'"`
	TransportCompany          *string    `gorm:"size:255"`
	VehicleNumber             *string    `gorm:"size:64"`
	DriverName                *string    `gorm:"size:255"`
	DriverContact             *string    `gorm:"size:64"`
	DepartureAt               *time.Time `gorm:"type:timestamptz"`
	ArrivalAt                 *time.Time `gorm:"type:timestamptz"`
	Cycle                     *string    `gorm:"size:64"`
	Status                    string     `gorm:"size:32;not null;default:'scheduled';index"`
	ApprovalStatus            string     `gorm:"size:32;not null;default:'pending';index"`
	DeliveryInstructions      *string    `gorm:"type:text"`
	Remarks                   *string    `gorm:"type:text"`
	CreatedBy                 *string    `gorm:"size:255"`
	ApprovedBy                *string    `gorm:"size:255"`
	ApprovedAt                *time.Time `gorm:"type:timestamptz"`
	CreatedAt                 time.Time  `gorm:"not null;default:now()"`
	UpdatedAt                 time.Time  `gorm:"not null;default:now()"`
	DeletedAt                 *time.Time `gorm:"index"`

	Items []ScheduleItemCustomer `gorm:"foreignKey:ScheduleID"`
}

func (ScheduleCustomer) TableName() string { return "delivery_schedules_customer" }

// ─── Delivery Schedule Items Customer ────────────────────────────────────────

type ScheduleItemCustomer struct {
	ID                           int64    `gorm:"primaryKey;autoIncrement"`
	UUID                         string   `gorm:"type:uuid;uniqueIndex;not null"`
	ScheduleID                   int64    `gorm:"not null;index"`
	CustomerOrderDocumentItemID  *int64   `gorm:"index"`
	LineNo                       int      `gorm:"not null"`
	ItemUniqCode                 string   `gorm:"size:100;not null;index"`
	Model                        *string  `gorm:"size:255"`
	PartName                     string   `gorm:"size:255;not null"`
	PartNumber                   string   `gorm:"size:128;not null"`
	TotalOrderQty                float64  `gorm:"type:numeric(15,4);not null;default:0"`
	TotalDeliveryQty             float64  `gorm:"type:numeric(15,4);not null;default:0"`
	UOM                          string   `gorm:"size:32;not null"`
	Cycle                        *string  `gorm:"size:64"`
	DNNumber                     *string  `gorm:"size:64;index"`
	Status                       string   `gorm:"size:32;not null;default:'scheduled';index"`
	FGReadinessStatus            string   `gorm:"size:32;not null;default:'unknown'"`
	CreatedAt                    time.Time `gorm:"not null;default:now()"`
	UpdatedAt                    time.Time `gorm:"not null;default:now()"`
}

func (ScheduleItemCustomer) TableName() string { return "delivery_schedule_items_customer" }

// ─── Delivery Note Customer ───────────────────────────────────────────────────

type DNCustomer struct {
	ID                      int64      `gorm:"primaryKey;autoIncrement"`
	UUID                    string     `gorm:"type:uuid;uniqueIndex;not null"`
	DNNumber                string     `gorm:"size:64;uniqueIndex;not null"`
	ScheduleID              *int64     `gorm:"index"`
	CustomerOrderDocumentID *int64     `gorm:"index"`
	CustomerOrderReference  *string    `gorm:"size:128"`
	CustomerID              int64      `gorm:"not null;index"`
	CustomerNameSnapshot    *string    `gorm:"size:255"`
	CustomerContactPerson   *string    `gorm:"size:255"`
	CustomerPhoneNumber     *string    `gorm:"size:64"`
	DeliveryAddress         *string    `gorm:"type:text"`
	DeliveryDate            time.Time  `gorm:"type:date;not null;index"`
	Priority                string     `gorm:"size:16;not null;default:'normal'"`
	TransportCompany        *string    `gorm:"size:255"`
	VehicleNumber           *string    `gorm:"size:64"`
	DriverName              *string    `gorm:"size:255"`
	DriverContact           *string    `gorm:"size:64"`
	DepartureAt             *time.Time `gorm:"type:timestamptz"`
	ArrivalAt               *time.Time `gorm:"type:timestamptz"`
	Status                  string     `gorm:"size:32;not null;default:'created';index"`
	ApprovalStatus          string     `gorm:"size:32;not null;default:'pending';index"`
	DeliveryInstructions    *string    `gorm:"type:text"`
	Remarks                 *string    `gorm:"type:text"`
	PrintedCount            int        `gorm:"not null;default:0"`
	CreatedBy               *string    `gorm:"size:255"`
	ApprovedBy              *string    `gorm:"size:255"`
	ApprovedAt              *time.Time `gorm:"type:timestamptz"`
	CreatedAt               time.Time  `gorm:"not null;default:now()"`
	UpdatedAt               time.Time  `gorm:"not null;default:now()"`

	Items []DNItemCustomer `gorm:"foreignKey:DNID"`
}

func (DNCustomer) TableName() string { return "delivery_notes_customer" }

// ─── Delivery Note Items Customer ─────────────────────────────────────────────

type DNItemCustomer struct {
	ID               int64      `gorm:"primaryKey;autoIncrement"`
	UUID             string     `gorm:"type:uuid;uniqueIndex;not null"`
	DNID             int64      `gorm:"not null;index"`
	ScheduleItemID   *int64     `gorm:"index"`
	LineNo           int        `gorm:"not null"`
	ItemUniqCode     string     `gorm:"size:100;not null;index"`
	Model            *string    `gorm:"size:255"`
	PartName         string     `gorm:"size:255;not null"`
	PartNumber       string     `gorm:"size:128;not null"`
	FGLocation       *string    `gorm:"size:64"`
	Quantity         float64    `gorm:"type:numeric(15,4);not null"`
	QtyShipped       float64    `gorm:"type:numeric(15,4);not null;default:0"`
	UOM              string     `gorm:"size:32;not null"`
	PackingNumber    string     `gorm:"size:100;uniqueIndex;not null"`
	QR               *string    `gorm:"type:text"`
	ShipmentStatus   string     `gorm:"size:32;not null;default:'created';index"`
	PrintedCount     int        `gorm:"not null;default:0"`
	ShippedAt        *time.Time `gorm:"type:timestamptz"`
	ShippedBy        *string    `gorm:"size:255"`
	CreatedAt        time.Time  `gorm:"not null;default:now()"`
	UpdatedAt        time.Time  `gorm:"not null;default:now()"`
}

func (DNItemCustomer) TableName() string { return "delivery_note_items_customer" }

// ─── Delivery Note Log Customer ───────────────────────────────────────────────

type DNLogCustomer struct {
	ID              int64     `gorm:"primaryKey;autoIncrement"`
	DNID            int64     `gorm:"not null;index"`
	DNItemID        int64     `gorm:"not null;index"`
	IdempotencyKey  *string   `gorm:"size:128;uniqueIndex"`
	ScanRef         *string   `gorm:"type:text"`
	ItemUniqCode    string    `gorm:"size:100;not null"`
	PackingNumber   *string   `gorm:"size:100;index"`
	ScanType        string    `gorm:"size:20;not null"`
	Qty             float64   `gorm:"type:numeric(15,2);not null"`
	FromLocation    *string   `gorm:"size:50"`
	ToLocation      *string   `gorm:"size:50"`
	ScannedBy       *string   `gorm:"size:255"`
	CreatedAt       time.Time `gorm:"not null;default:now()"`
}

func (DNLogCustomer) TableName() string { return "delivery_note_logs_customer" }

// ─── Pagination ───────────────────────────────────────────────────────────────

type Pagination struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"total_pages"`
}
