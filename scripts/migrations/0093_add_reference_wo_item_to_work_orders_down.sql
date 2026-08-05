-- =============================================================================
-- Migration: 0093_add_reference_wo_item_to_work_orders_down.sql
-- =============================================================================

ALTER TABLE IF EXISTS work_orders
    DROP COLUMN IF EXISTS reference_wo_item_id;
