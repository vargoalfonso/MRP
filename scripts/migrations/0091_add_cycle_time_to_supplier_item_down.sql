-- Revert cycle_time column from supplier_item
ALTER TABLE supplier_item
DROP COLUMN IF EXISTS cycle_time;
