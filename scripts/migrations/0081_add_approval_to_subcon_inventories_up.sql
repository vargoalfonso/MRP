-- Migration: 0081_add_approval_to_subcon_inventories_up.sql
-- Add approval workflow (status-only) + auto-source tracking to subcon_inventories.
-- approval_status: pending | approved | rejected. Existing rows default to 'approved'
-- so previously-created manual stock is not hidden by the new approval gate.

ALTER TABLE subcon_inventories
ADD COLUMN IF NOT EXISTS approval_status varchar(32) NOT NULL DEFAULT 'approved',
ADD COLUMN IF NOT EXISTS approved_by varchar(255),
ADD COLUMN IF NOT EXISTS approved_at timestamptz,
ADD COLUMN IF NOT EXISTS source_type varchar(32),
ADD COLUMN IF NOT EXISTS source_ref varchar(255);

-- New auto-synced rows default to 'pending' (handled in application code).

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'subcon_inventories_approval_status_ck'
    ) THEN
        ALTER TABLE subcon_inventories
        ADD CONSTRAINT subcon_inventories_approval_status_ck
        CHECK (approval_status IN ('pending', 'approved', 'rejected'));
    END IF;
END $$;

-- Idempotency for auto-sync: one subcon row per (source_type, source_ref).
CREATE UNIQUE INDEX IF NOT EXISTS subcon_inventories_source_uq
ON subcon_inventories (source_type, source_ref)
WHERE source_type IS NOT NULL AND source_ref IS NOT NULL AND deleted_at IS NULL;
