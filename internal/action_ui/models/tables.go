package models

import "time"

// IncomingReceivingScan maps to incoming_receiving_scans (append-only).
// Created by migration: scripts/migrations/0015_dn_feature_up.sql
type IncomingReceivingScan struct {
	ID               int64     `gorm:"column:id;primaryKey;autoIncrement"`
	IncomingDNItemID string    `gorm:"column:incoming_dn_item_id;not null"`
	IdempotencyKey   *string   `gorm:"column:idempotency_key"`
	ScanRef          string    `gorm:"column:scan_ref;not null"`
	Qty              float64   `gorm:"column:qty;type:numeric(15,4);not null"`
	WeightKg         *float64  `gorm:"column:weight_kg;type:numeric(15,4)"`
	ScannedAt        time.Time `gorm:"column:scanned_at;not null;default:now()"`
	ScannedBy        *string   `gorm:"column:scanned_by"`
}

func (IncomingReceivingScan) TableName() string { return "incoming_receiving_scans" }
