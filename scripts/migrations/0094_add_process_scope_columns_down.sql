-- 0094_add_process_scope_columns_down.sql

DROP INDEX IF EXISTS idx_raw_material_logs_step_seq;
DROP INDEX IF EXISTS idx_raw_material_logs_process_name;

ALTER TABLE raw_material_logs DROP COLUMN IF EXISTS step_seq;
ALTER TABLE raw_material_logs DROP COLUMN IF EXISTS process_name;

ALTER TABLE wip_items DROP COLUMN IF EXISTS from_process;
