-- =============================================================================
-- Migration: 0016_extend_po_split_settings_up.sql
-- Module   : PO Budget
-- Purpose  : Extend po_split_settings to support UI fields:
--            min_order_qty, max_split_lines, split_rule, status
-- Notes    : Backward compatible (adds nullable columns).
-- =============================================================================

ALTER TABLE IF EXISTS po_split_settings
    ADD COLUMN IF NOT EXISTS min_order_qty int,
    ADD COLUMN IF NOT EXISTS max_split_lines int,
    ADD COLUMN IF NOT EXISTS split_rule varchar(64),
    ADD COLUMN IF NOT EXISTS status varchar(20) NOT NULL DEFAULT 'Active'
        CHECK (status IN ('Active','Inactive'));

CREATE INDEX IF NOT EXISTS idx_po_split_settings_status ON po_split_settings (status);
