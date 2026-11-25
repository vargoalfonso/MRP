-- Rollback 0028
ALTER TABLE purchase_orders
ADD COLUMN IF NOT EXISTS expected_delivery_date date;