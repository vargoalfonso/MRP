-- =============================================================================
-- Migration: 0018_procurement_po_ext_down.sql
-- Rolls back 0018_procurement_po_ext_up.sql
--
-- NOTE: Does NOT drop purchase_orders / purchase_order_items / supplier because
--       those may have been created by the legacy system; only the columns/tables
--       added by *this* migration are removed.
-- =============================================================================

DROP TABLE IF EXISTS purchase_order_logs;

ALTER TABLE IF EXISTS purchase_order_items
    DROP COLUMN IF EXISTS po_budget_entry_id;

ALTER TABLE IF EXISTS purchase_orders
    DROP COLUMN IF EXISTS po_budget_entry_id,
    DROP COLUMN IF EXISTS po_stage;

DROP INDEX IF EXISTS idx_purchase_orders_po_stage;
DROP INDEX IF EXISTS idx_purchase_orders_po_budget_entry;
DROP INDEX IF EXISTS idx_purchase_orders_type_period_stage;
DROP INDEX IF EXISTS idx_po_items_budget_entry;
