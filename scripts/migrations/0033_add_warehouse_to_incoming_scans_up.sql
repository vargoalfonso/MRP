-- Migration: 0033_add_warehouse_to_incoming_scans_up.sql
-- Adds warehouse_location column to incoming_receiving_scans so operators can
-- specify the destination warehouse at scan time, which then flows through to
-- inventory (raw_materials / indirect_raw_materials) on QC approval.

ALTER TABLE incoming_receiving_scans
ADD COLUMN IF NOT EXISTS warehouse_location VARCHAR(255) NULL;