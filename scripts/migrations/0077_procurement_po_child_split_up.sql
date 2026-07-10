-- =============================================================================
-- Migration: 0077_procurement_po_child_split_up.sql
-- Feature  : Procurement PO — split PO Budget children into PO1/PO2 lines
--
-- When a raw_material/indirect PO Budget entry is pulled into procurement, each
-- child material in po_budget_entries.detail_jsonb is split into PO1 and PO2
-- lines (childQty * po1_pct / 100 and childQty * po2_pct / 100). We persist BOTH:
--   1. Real per-child rows in purchase_order_items (with material_spec snapshot).
--   2. A per-stage detail_jsonb snapshot on purchase_orders.
--
-- Changes:
--   1. purchase_orders.detail_jsonb  — snapshot of split children for that stage
--   2. purchase_order_items.material_spec  — child material spec snapshot
--   3. purchase_order_items.child_uniq_code — child uniq (distinct from parent)
--   4. purchase_order_items.material_grade  — quick-access grade for filtering
-- =============================================================================

ALTER TABLE purchase_orders
    ADD COLUMN IF NOT EXISTS detail_jsonb jsonb NOT NULL DEFAULT '{}'::jsonb;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'purchase_orders_detail_jsonb_is_object_ck'
    ) THEN
        ALTER TABLE purchase_orders
        ADD CONSTRAINT purchase_orders_detail_jsonb_is_object_ck
        CHECK (jsonb_typeof(detail_jsonb) = 'object');
    END IF;
END $$;

ALTER TABLE purchase_order_items
    ADD COLUMN IF NOT EXISTS material_spec   jsonb,
    ADD COLUMN IF NOT EXISTS child_uniq_code varchar(64),
    ADD COLUMN IF NOT EXISTS material_grade  varchar(100);

CREATE INDEX IF NOT EXISTS idx_po_items_child_uniq
    ON purchase_order_items (child_uniq_code);
