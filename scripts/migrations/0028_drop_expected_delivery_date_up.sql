-- Migration 0028: drop expected_delivery_date from purchase_orders
ALTER TABLE purchase_orders
DROP COLUMN IF EXISTS expected_delivery_date;