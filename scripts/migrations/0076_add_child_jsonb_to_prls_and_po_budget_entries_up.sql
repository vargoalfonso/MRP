-- Migration: 0076_add_child_jsonb_to_prls_and_po_budget_entries_up.sql
-- Add lightweight child references to PRLs and parent+children detail snapshots to PO Budget entries.

ALTER TABLE prls
ADD COLUMN IF NOT EXISTS child_jsonb jsonb NOT NULL DEFAULT '[]'::jsonb;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'prls_child_jsonb_is_array_ck'
    ) THEN
        ALTER TABLE prls
        ADD CONSTRAINT prls_child_jsonb_is_array_ck
        CHECK (jsonb_typeof(child_jsonb) = 'array');
    END IF;
END $$;

ALTER TABLE po_budget_entries
ADD COLUMN IF NOT EXISTS detail_jsonb jsonb NOT NULL DEFAULT '{}'::jsonb;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'po_budget_entries_detail_jsonb_is_object_ck'
    ) THEN
        ALTER TABLE po_budget_entries
        ADD CONSTRAINT po_budget_entries_detail_jsonb_is_object_ck
        CHECK (jsonb_typeof(detail_jsonb) = 'object');
    END IF;
END $$;
