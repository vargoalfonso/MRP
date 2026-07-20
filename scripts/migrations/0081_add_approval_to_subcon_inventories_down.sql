-- Migration: 0081_add_approval_to_subcon_inventories_down.sql

DROP INDEX IF EXISTS subcon_inventories_source_uq;

ALTER TABLE subcon_inventories
DROP CONSTRAINT IF EXISTS subcon_inventories_approval_status_ck;

ALTER TABLE subcon_inventories
DROP COLUMN IF EXISTS source_ref,
DROP COLUMN IF EXISTS source_type,
DROP COLUMN IF EXISTS approved_at,
DROP COLUMN IF EXISTS approved_by,
DROP COLUMN IF EXISTS approval_status;
