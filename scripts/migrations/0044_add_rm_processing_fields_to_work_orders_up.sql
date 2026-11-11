-- =============================================================================
-- Migration: 0044_add_rm_processing_fields_to_work_orders_up.sql
-- Feature  : Store RM processing fields directly in work_orders (single-table design)
-- Notes    : Keep rm_processing_work_orders as legacy table (no drop here).
-- =============================================================================

ALTER TABLE IF EXISTS work_orders
ADD COLUMN IF NOT EXISTS source_material_uniq VARCHAR(64),
ADD COLUMN IF NOT EXISTS target_material_uniq VARCHAR(64),
ADD COLUMN IF NOT EXISTS model VARCHAR(128),
ADD COLUMN IF NOT EXISTS grade_size VARCHAR(255),
ADD COLUMN IF NOT EXISTS input_qty NUMERIC(15, 4),
ADD COLUMN IF NOT EXISTS input_uom VARCHAR(32),
ADD COLUMN IF NOT EXISTS output_qty NUMERIC(15, 4),
ADD COLUMN IF NOT EXISTS output_uom VARCHAR(32),
ADD COLUMN IF NOT EXISTS date_issued DATE,
ADD COLUMN IF NOT EXISTS date_completed DATE,
ADD COLUMN IF NOT EXISTS cycle_time_days INT,
ADD COLUMN IF NOT EXISTS remarks TEXT;

CREATE INDEX IF NOT EXISTS idx_work_orders_source_material_uniq ON work_orders (source_material_uniq);

CREATE INDEX IF NOT EXISTS idx_work_orders_target_material_uniq ON work_orders (target_material_uniq);

CREATE INDEX IF NOT EXISTS idx_work_orders_date_issued ON work_orders (date_issued);

-- Best-effort backfill from legacy rm_processing_work_orders if table exists.
DO $$
BEGIN
    IF to_regclass('public.rm_processing_work_orders') IS NOT NULL THEN
        UPDATE work_orders wo
        SET
            source_material_uniq = rm.source_material_uniq,
            target_material_uniq = rm.target_material_uniq,
            model                = rm.model,
            grade_size           = rm.grade_size,
            input_qty            = rm.input_qty,
            input_uom            = rm.input_uom,
            output_qty           = rm.output_qty,
            output_uom           = rm.output_uom,
            date_issued          = rm.date_issued,
            date_completed       = rm.date_completed,
            cycle_time_days      = rm.cycle_time_days,
            remarks              = rm.remarks
        FROM rm_processing_work_orders rm
        WHERE rm.work_order_id = wo.id
          AND wo.wo_kind = 'rm_processing'
          AND wo.source_material_uniq IS NULL;
    END IF;
END $$;