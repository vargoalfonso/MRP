-- Migration: 0033_add_warehouse_to_incoming_scans_down.sql
ALTER TABLE incoming_receiving_scans
DROP COLUMN IF EXISTS warehouse_location;