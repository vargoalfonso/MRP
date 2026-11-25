-- =============================================================================
-- Migration: 0035_create_work_orders_down.sql
-- Rollback : Drop production WO tables and qc_tasks extensions
-- =============================================================================

-- Remove qc_tasks FKs and columns (keep table for incoming QC)
ALTER TABLE IF EXISTS qc_tasks
    DROP CONSTRAINT IF EXISTS qc_tasks_source_scan_id_fk;

ALTER TABLE IF EXISTS qc_tasks
    DROP CONSTRAINT IF EXISTS qc_tasks_wo_item_id_fk;

ALTER TABLE IF EXISTS qc_tasks
    DROP CONSTRAINT IF EXISTS qc_tasks_wo_id_fk;

ALTER TABLE IF EXISTS qc_tasks
    DROP COLUMN IF EXISTS source_scan_id,
    DROP COLUMN IF EXISTS wo_item_id,
    DROP COLUMN IF EXISTS wo_id;

DROP TABLE IF EXISTS production_scan_logs;
DROP TABLE IF EXISTS work_order_items;
DROP TABLE IF EXISTS work_orders;
