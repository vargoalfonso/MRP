-- =============================================================================
-- Migration: 0077_procurement_po_child_split_down.sql
-- Reverts 0077_procurement_po_child_split_up.sql
-- =============================================================================

DROP INDEX IF EXISTS idx_po_items_child_uniq;

ALTER TABLE purchase_order_items
    DROP COLUMN IF EXISTS material_spec,
    DROP COLUMN IF EXISTS child_uniq_code,
    DROP COLUMN IF EXISTS material_grade;

ALTER TABLE purchase_orders
    DROP CONSTRAINT IF EXISTS purchase_orders_detail_jsonb_is_object_ck;

ALTER TABLE purchase_orders
    DROP COLUMN IF EXISTS detail_jsonb;
