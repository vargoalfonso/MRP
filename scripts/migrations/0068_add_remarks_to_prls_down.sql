-- Rollback for 0068_add_remarks_to_prls_up.sql
ALTER TABLE prls
DROP COLUMN IF EXISTS remarks;
