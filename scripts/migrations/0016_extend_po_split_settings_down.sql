-- =============================================================================
-- Migration: 0016_extend_po_split_settings_down.sql
-- =============================================================================

DROP INDEX IF EXISTS idx_po_split_settings_status;

ALTER TABLE IF EXISTS po_split_settings
    DROP COLUMN IF EXISTS min_order_qty,
    DROP COLUMN IF EXISTS max_split_lines,
    DROP COLUMN IF EXISTS split_rule,
    DROP COLUMN IF EXISTS status;
