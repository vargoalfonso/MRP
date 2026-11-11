package models

import "time"

type MasterMachine struct {
	ID              int64     `gorm:"primaryKey" json:"id"`
	MachineNumber   string    `gorm:"column:machine_number" json:"machine_number"`
	MachineName     string    `gorm:"column:machine_name" json:"machine_name"`
	ProductionLine  string    `gorm:"column:production_line" json:"production_line"`
	ProcessID       *int64    `gorm:"column:process_id" json:"process_id"`
	MachineCapacity *int      `gorm:"column:machine_capacity" json:"machine_capacity"`
	Status          string    `gorm:"column:status" json:"status"`
	QRImageBase64   *string   `gorm:"column:qr_image_base64;type:text" json:"qr_image_base64"`
	CreatedAt       time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (MasterMachine) TableName() string { return "master_machines" }
