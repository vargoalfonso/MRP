-- =============================================================================
-- Migration: 0044_add_rm_processing_fields_to_work_orders_down.sql
-- Rollback : Remove RM processing fields from work_orders
-- =============================================================================

DROP INDEX IF EXISTS idx_work_orders_date_issued;

DROP INDEX IF EXISTS idx_work_orders_target_material_uniq;

DROP INDEX IF EXISTS idx_work_orders_source_material_uniq;

ALTER TABLE IF EXISTS work_orders
DROP COLUMN IF EXISTS remarks,
DROP COLUMN IF EXISTS cycle_time_days,
DROP COLUMN IF EXISTS date_completed,
DROP COLUMN IF EXISTS date_issued,
DROP COLUMN IF EXISTS output_uom,
DROP COLUMN IF EXISTS output_qty,
DROP COLUMN IF EXISTS input_uom,
DROP COLUMN IF EXISTS input_qty,
DROP COLUMN IF EXISTS grade_size,
DROP COLUMN IF EXISTS model,
DROP COLUMN IF EXISTS target_material_uniq,
DROP COLUMN IF EXISTS source_material_uniq;