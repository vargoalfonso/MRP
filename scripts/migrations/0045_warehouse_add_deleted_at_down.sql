-- Down migration for 0045.
-- WARNING: Dropping deleted_at may break GORM soft-delete queries.

DROP INDEX IF EXISTS public.idx_warehouse_deleted_at;

ALTER TABLE IF EXISTS public.warehouse
    DROP COLUMN IF EXISTS deleted_at;
