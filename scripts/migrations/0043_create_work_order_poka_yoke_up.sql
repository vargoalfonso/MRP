-- =============================================================================
-- Migration: 0043_create_work_order_poka_yoke_up.sql
-- Feature  : Poka Yoke snapshot directly on work_order_items
-- Notes    : Keep scan logic lightweight: no BOM pull at runtime.
--            This migration extends existing work_order_items (no new table).
-- =============================================================================

ALTER TABLE IF EXISTS work_order_items
ADD COLUMN IF NOT EXISTS process_flow_json JSONB NOT NULL DEFAULT '[]'::jsonb,
ADD COLUMN IF NOT EXISTS current_step_seq INT NOT NULL DEFAULT 1,
ADD COLUMN IF NOT EXISTS last_scanned_process VARCHAR(64),
ADD COLUMN IF NOT EXISTS scan_in_count INT NOT NULL DEFAULT 0,
ADD COLUMN IF NOT EXISTS scan_out_count INT NOT NULL DEFAULT 0,
ADD COLUMN IF NOT EXISTS total_good_qty NUMERIC(15, 4) NOT NULL DEFAULT 0,
ADD COLUMN IF NOT EXISTS total_ng_qty NUMERIC(15, 4) NOT NULL DEFAULT 0,
ADD COLUMN IF NOT EXISTS total_scrap_qty NUMERIC(15, 4) NOT NULL DEFAULT 0;

-- PostgreSQL does not support ADD CONSTRAINT IF NOT EXISTS directly.
DO $$
BEGIN
    IF to_regclass('public.work_order_items') IS NOT NULL THEN
        IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'work_order_items_flow_array_ck') THEN
            ALTER TABLE work_order_items
                ADD CONSTRAINT work_order_items_flow_array_ck
                CHECK (jsonb_typeof(process_flow_json) = 'array');
        END IF;

        IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'work_order_items_current_step_ck') THEN
            ALTER TABLE work_order_items
                ADD CONSTRAINT work_order_items_current_step_ck
                CHECK (current_step_seq >= 1);
        END IF;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_wo_items_current_step ON work_order_items (current_step_seq);

CREATE INDEX IF NOT EXISTS idx_wo_items_last_scanned_process ON work_order_items (last_scanned_process);

CREATE INDEX IF NOT EXISTS idx_wo_items_process_flow_gin ON work_order_items USING GIN (process_flow_json);