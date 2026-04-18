package models

import (
	"time"
)

type DeliveryNote struct {
	ID       int64  `json:"id" gorm:"primaryKey"`              //request
	DNNumber string `json:"dn_number" gorm:"type:varchar(50)"` //request
	// CustomerID      int64     `json:"customer_id"`                                         //request
	// ContactPerson   string    `json:"contact_person"`                                      //request
	Period          string    `json:"period"`                                              //request
	PONumber        string    `json:"po_number"`                                           //request diambil dari po_number check po ini ada atau tidak, kalau tidak ada maka error
	Type            string    `json:"type"`                                                //request
	Status          string    `json:"status"`                                              //draft, incoming, completed. default draft. nanti kalau semua item sudah diterima maka status jadi completed
	SupplierID      int64     `json:"supplier_id"`                                         //request diambil dari supplier_id di purchase order
	Supplier        Supplier  `json:"supplier" gorm:"foreignKey:SupplierID;references:ID"` //relasi ke supplier untuk mendapatkan nama supplier
	TotalPOQty      int64     `json:"total_po_qty"`                                        //request diambil dari total qty di purchase order items bedasarkan po_id
	TotalPOIncoming int64     `json:"total_po_incoming"`                                   //request diambil dari total qty yang sudah diterima di delivery note items bedasarkan po_id
	TotalDNCreated  int64     `json:"total_dn_created"`                                    //request total data dari purchaase order items dari po_id yang sudah dibuat dn nya (bedasarkan po_number)
	TotalDNIncoming int64     `json:"total_dn_incoming"`                                   //request total data dari purchaase order items dari po_id yang sudah diterima barangnya (bedasarkan po_number)
	CreatedBy       string    `json:"created_by"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	Items []DeliveryNoteItem `json:"items" gorm:"foreignKey:DNID;references:ID"`
}

type Supplier struct {
	ID   int64  `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	UUID string `json:"uuid" gorm:"column:uuid;type:uuid;not null"`

	SupplierCode  string `json:"supplier_code" gorm:"column:supplier_code;size:50;not null"`
	SupplierName  string `json:"supplier_name" gorm:"column:supplier_name;size:255;not null"`
	ContactPerson string `json:"contact_person" gorm:"column:contact_person;size:255;not null"`
	ContactNumber string `json:"contact_number" gorm:"column:contact_number;size:50;not null"`
	EmailAddress  string `json:"email_address" gorm:"column:email_address;size:255;not null"`

	MaterialCategory *string `json:"material_category,omitempty" gorm:"column:material_category;size:50"`

	FullAddress string `json:"full_address" gorm:"column:full_address;type:text;not null"`
	City        string `json:"city" gorm:"column:city;size:150;not null"`
	Province    string `json:"province" gorm:"column:province;size:150;not null"`
	Country     string `json:"country" gorm:"column:country;size:150;not null"`

	TaxIDNPWP string `json:"tax_id_npwp" gorm:"column:tax_id_npwp;size:50;not null"`

	BankName          string `json:"bank_name" gorm:"column:bank_name;size:150;not null"`
	BankAccountNumber string `json:"bank_account_number" gorm:"column:bank_account_number;size:100;not null"`
	BankAccountName   string `json:"bank_account_name" gorm:"column:bank_account_name;size:255;not null"`

	PaymentTerms         string `json:"payment_terms" gorm:"column:payment_terms;size:150;not null"`
	DeliveryLeadTimeDays int32  `json:"delivery_lead_time_days" gorm:"column:delivery_lead_time_days;default:0"`

	Status string `json:"status" gorm:"column:status;size:20;default:Active"`

	CreatedAt time.Time  `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" gorm:"column:deleted_at"`
}

type DeliveryNoteItem struct {
	ID             int64           `json:"id" gorm:"primaryKey;column:id"`
	DNID           int64           `json:"dn_id" gorm:"column:dn_id;index"`                 //foreign key ke delivery note
	ItemUniqCode   string          `json:"item_uniq_code" gorm:"column:item_uniq_code"`     //request, diambil dari item_uniq_code di purchase order items bedasarkan po_id
	Quantity       int64           `json:"quantity" gorm:"column:quantity"`                 //request, diambil dari ordered_qty di purchase order items bedasarkan po_id dan item_uniq_code
	UOM            string          `json:"uom" gorm:"column:uom"`                           //request, diambil dari uom di purchase order items bedasarkan po_id dan item_uniq_code
	Weight         int64           `json:"weight" gorm:"column:weight"`                     //request, diambil dari weight di purchase order items bedasarkan po_id dan item_uniq_code
	KanbanID       int64           `json:"kanban_id" gorm:"column:kanban_id"`               //request, diambil dari table kanban_parameter bedasarkan item_uniq_code
	Kanban         KanbanParameter `json:"kanban" gorm:"foreignKey:KanbanID;references:ID"` //relasi ke kanban parameter untuk mendapatkan kanban number dan pcs per kanban
	QR             string          `json:"qr" gorm:"column:qr"`                             //hasil generate qr code yang berisi dn_number dan item_uniq_code, format datanya dn_number-item_uniq_code
	CreatedAt      time.Time       `json:"created_at" gorm:"column:created_at"`
	UpdatedAt      time.Time       `json:"updated_at" gorm:"column:updated_at"`
	OrderQty       int64           `json:"order_qty" gorm:"column:order_qty"`             //request, diambil dari ordered_qty di purchase order items bedasarkan po_id dan item_uniq_code
	DateIncoming   *time.Time      `json:"date_incoming" gorm:"column:date_incoming"`     //request, diisi ketika barang diterima
	QtyStated      int64           `json:"qty_stated" gorm:"column:qty_stated"`           //request, diisi ketika barang diterima, diambil dari quantity di delivery note item
	QtyReceived    int64           `json:"qty_received" gorm:"column:qty_received"`       //request, diisi ketika barang diterima, diambil dari quantity yang diterima di lapangan, bisa lebih kecil atau lebih besar dari qty_stated
	WeightReceived float64         `json:"weight_received" gorm:"column:weight_received"` //request, diisi ketika barang diterima, diambil dari weight yang diterima di lapangan, bisa lebih kecil atau lebih besar dari weight di delivery note item
	QualityStatus  string          `json:"quality_status" gorm:"column:quality_status"`   //request, diisi ketika barang diterima, bisa bernilai "good" atau "damaged"
	PcsPerKanban   int64           `json:"pcs_per_kanban" gorm:"column:pcs_per_kanban"`   //request, diambil dari pcs_per_kanban di purchase order items bedasarkan po_id dan item_uniq_code
	ReceivedAt     *time.Time      `json:"received_at" gorm:"column:received_at"`         //request, diisi ketika barang diterima, berisi tanggal dan jam ketika barang diterima
	PackingNumber  string          `json:"packing_number" gorm:"column:packing_number"`   //request, diambil dari kanban number di kanban parameter bedasarkan item_uniq_code
	Check          string          `json:"check" gorm:"check"`                            //field untuk menampung nilai check ketika menerima barang,
	QtySent        int64           `json:"qty_sent" gorm:"qty_sent"`
}

type KanbanParameter struct {
	ID           int64     `json:"id" gorm:"primaryKey"`
	KanbanNumber string    `json:"kanban_number"`
	ItemUniqCode string    `json:"item_uniq_code"`
	KanbanQty    int       `json:"kanban_qty"`
	MinStock     int       `json:"min_stock"`
	MaxStock     int       `json:"max_stock"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type PurchaseOrder struct {
	PoID                 int64      `json:"po_id" db:"po_id"`
	PoType               string     `json:"po_type" db:"po_type"`
	Period               string     `json:"period" db:"period"`
	PoNumber             string     `json:"po_number" db:"po_number"`
	PoBudgetID           int64      `json:"po_budget_id" db:"po_budget_id"`
	SupplierID           int64      `json:"supplier_id" db:"supplier_id"`
	Status               string     `json:"status" db:"status"`
	CreatedAt            time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at" db:"updated_at"`
	PoDate               *time.Time `json:"po_date" db:"po_date"`
	ExpectedDeliveryDate *time.Time `json:"expected_delivery_date" db:"expected_delivery_date"`
	Currency             string     `json:"currency" db:"currency"`
	TotalAmount          float64    `json:"total_amount" db:"total_amount"`
	ExternalSystem       string     `json:"external_system" db:"external_system"`
	ExternalPONumber     string     `json:"external_po_number" db:"external_po_number"`
	CreatedBy            string     `json:"created_by" db:"created_by"`
	UpdatedBy            string     `json:"updated_by" db:"updated_by"`
	PoStage              int64      `json:"po_stage" db:"po_stage"`
	PoBudgetEntryID      int64      `json:"po_budget_entry_id" db:"po_budget_entry_id"`
	TotalWeight          float64    `json:"total_weight" db:"total_weight"`
}

type PurchaseOrderItem struct {
	ID              int64     `json:"id" db:"id"`
	PoID            int64     `json:"po_id" db:"po_id"`
	LineNo          int64     `json:"line_no" db:"line_no"`
	ItemUniqCode    string    `json:"item_uniq_code" db:"item_uniq_code"`
	ProductModel    string    `json:"product_model" db:"product_model"`
	MaterialType    string    `json:"material_type" db:"material_type"`
	PartName        string    `json:"part_name" db:"part_name"`
	PartNumber      string    `json:"part_number" db:"part_number"`
	UOM             string    `json:"uom" db:"uom"`
	WeightKg        float64   `json:"weight_kg" db:"weight_kg"`
	Description     string    `json:"description" db:"description"`
	OrderedQty      float64   `json:"ordered_qty" db:"ordered_qty"`
	UnitPrice       float64   `json:"unit_price" db:"unit_price"`
	Amount          float64   `json:"amount" db:"amount"`
	PcsPerKanban    int64     `json:"pcs_per_kanban" db:"pcs_per_kanban"`
	Status          string    `json:"status" db:"status"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
	PoBudgetEntryID int64     `json:"po_budget_entry_id" db:"po_budget_entry_id"`
}

type DeliveryNoteLog struct {
	ID            int64     `json:"id" gorm:"primaryKey;column:id"`
	DNID          int64     `json:"dn_id" gorm:"column:dn_id;not null;index"`
	DNItemID      int64     `json:"dn_item_id" gorm:"column:dn_item_id;not null;index"`
	ItemUniqCode  string    `json:"item_uniq_code" gorm:"column:item_uniq_code;type:varchar(100);not null"`
	PackingNumber string    `json:"packing_number" gorm:"column:packing_number;type:varchar(100);index"`
	ScanType      string    `json:"scan_type" gorm:"column:scan_type;type:varchar(20);not null"` // outgoing | incoming
	Qty           float64   `json:"qty" gorm:"column:qty;type:numeric(15,2);not null"`
	FromLocation  string    `json:"from_location" gorm:"column:from_location;type:varchar(50)"`
	ToLocation    string    `json:"to_location" gorm:"column:to_location;type:varchar(50)"`
	CreatedAt     time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
}
