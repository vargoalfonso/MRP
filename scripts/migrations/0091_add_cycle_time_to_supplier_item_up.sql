-- Add cycle_time column (in days) to supplier_item
ALTER TABLE supplier_item
ADD COLUMN IF NOT EXISTS cycle_time BIGINT NULL;
