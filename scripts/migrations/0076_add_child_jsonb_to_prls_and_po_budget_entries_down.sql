-- Migration: 0076_add_child_jsonb_to_prls_and_po_budget_entries_down.sql

ALTER TABLE po_budget_entries
DROP CONSTRAINT IF EXISTS po_budget_entries_detail_jsonb_is_object_ck;

ALTER TABLE po_budget_entries
DROP COLUMN IF EXISTS detail_jsonb;

ALTER TABLE prls
DROP CONSTRAINT IF EXISTS prls_child_jsonb_is_array_ck;

ALTER TABLE prls
DROP COLUMN IF EXISTS child_jsonb;
