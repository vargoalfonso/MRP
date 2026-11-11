-- Add soft-delete support to warehouse table (GORM uses deleted_at in queries).

ALTER TABLE IF EXISTS public.warehouse
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ NULL;

CREATE INDEX IF NOT EXISTS idx_warehouse_deleted_at ON public.warehouse (deleted_at);
