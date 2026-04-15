-- =============================================================================
-- Migration: 0043_create_work_order_poka_yoke_down.sql
-- Rollback : Remove Poka Yoke columns from work_order_items
-- =============================================================================

DROP INDEX IF EXISTS idx_wo_items_process_flow_gin;

DROP INDEX IF EXISTS idx_wo_items_last_scanned_process;

DROP INDEX IF EXISTS idx_wo_items_current_step;

ALTER TABLE IF EXISTS work_order_items
DROP CONSTRAINT IF EXISTS work_order_items_flow_array_ck;

ALTER TABLE IF EXISTS work_order_items
DROP CONSTRAINT IF EXISTS work_order_items_current_step_ck;

ALTER TABLE IF EXISTS work_order_items
DROP COLUMN IF EXISTS total_scrap_qty,
DROP COLUMN IF EXISTS total_ng_qty,
DROP COLUMN IF EXISTS total_good_qty,
DROP COLUMN IF EXISTS scan_out_count,
DROP COLUMN IF EXISTS scan_in_count,
DROP COLUMN IF EXISTS last_scanned_process,
DROP COLUMN IF EXISTS current_step_seq,
DROP COLUMN IF EXISTS process_flow_json;