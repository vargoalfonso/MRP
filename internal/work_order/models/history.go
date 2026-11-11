package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type WorkOrderImportHistory struct {
	ID            int64          `gorm:"primaryKey"                json:"id"`
	FileName      string         `gorm:"type:varchar(255)"         json:"file_name"`
	FileSizeKb    int            `                                 json:"file_size_kb"`
	RowCount      int            `                                 json:"row_count"`
	UploadedBy    string         `gorm:"type:varchar(255)"         json:"uploaded_by"`
	Status        string         `gorm:"type:varchar(20)"          json:"status"` // success | partial | error
	Summary       string         `gorm:"type:text"                 json:"summary"`
	ImportedCount int            `                                 json:"imported_count"`
	FailedCount   int            `                                 json:"failed_count"`
	RequestID     string         `gorm:"type:varchar(64)"          json:"request_id"`
	ErrorFile     []byte         `gorm:"type:bytea"                json:"-"`              // xlsx byte, tidak ikut di list
	HasErrorFile  bool           `gorm:"->;column:has_error_file"  json:"has_error_file"` // read-only, hasil hitung SQL
	CreatedAt     time.Time      `                                 json:"created_at"`
	UpdatedAt     time.Time      `                                 json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index"                     json:"-"`
	PreviewRows   datatypes.JSON `gorm:"column:preview_rows;type:jsonb" json:"preview_rows,omitempty"`
}

func (WorkOrderImportHistory) TableName() string { return "work_order_import_histories" }
