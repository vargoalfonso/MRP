-- Sync existing public.warehouse table to match the API model.
--
-- Why: some DBs already have a "warehouse" table with a different schema
-- (e.g., only {id, code, deleted_at}). The original migration
-- scripts/migrations/0035_create_warehouse_up.sql uses CREATE TABLE IF NOT EXISTS,
-- so it won't update an already-existing table.
--
-- This migration is written to be additive and safe for existing data.

BEGIN;

-- Needed for gen_random_uuid() on Postgres.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Core columns expected by the API.
ALTER TABLE IF EXISTS public.warehouse
    ADD COLUMN IF NOT EXISTS uuid UUID,
    ADD COLUMN IF NOT EXISTS warehouse_name VARCHAR(255),
    ADD COLUMN IF NOT EXISTS type_warehouse VARCHAR(50),
    ADD COLUMN IF NOT EXISTS plant_id VARCHAR(100),
    ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ;

-- Backfill existing rows so we can enforce NOT NULL.
UPDATE public.warehouse
SET
    uuid = COALESCE(uuid, gen_random_uuid()),
    warehouse_name = COALESCE(warehouse_name, NULLIF(code, ''), 'UNKNOWN'),
    type_warehouse = COALESCE(type_warehouse, 'general'),
    plant_id = COALESCE(plant_id, 'UNKNOWN'),
    created_at = COALESCE(created_at, NOW()),
    updated_at = COALESCE(updated_at, NOW())
WHERE
    uuid IS NULL
    OR warehouse_name IS NULL
    OR type_warehouse IS NULL
    OR plant_id IS NULL
    OR created_at IS NULL
    OR updated_at IS NULL;

-- Enforce not-null for new columns.
ALTER TABLE public.warehouse
    ALTER COLUMN uuid SET NOT NULL,
    ALTER COLUMN warehouse_name SET NOT NULL,
    ALTER COLUMN type_warehouse SET NOT NULL,
    ALTER COLUMN plant_id SET NOT NULL,
    ALTER COLUMN created_at SET NOT NULL,
    ALTER COLUMN updated_at SET NOT NULL;

-- Constraints and indexes matching 0035_create_warehouse_up.sql.
DO $$
BEGIN
    -- Unique UUID.
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'warehouse_uuid_key'
    ) THEN
        ALTER TABLE public.warehouse
            ADD CONSTRAINT warehouse_uuid_key UNIQUE (uuid);
    END IF;

    -- Unique warehouse_name per plant.
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'warehouse_name_plant_key'
    ) THEN
        ALTER TABLE public.warehouse
            ADD CONSTRAINT warehouse_name_plant_key UNIQUE (warehouse_name, plant_id);
    END IF;

    -- Type check.
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'warehouse_type_check'
    ) THEN
        ALTER TABLE public.warehouse
            ADD CONSTRAINT warehouse_type_check CHECK (type_warehouse IN ('raw_material', 'wip', 'finished_goods', 'subcon', 'general'));
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_warehouse_deleted_at ON public.warehouse (deleted_at);
CREATE INDEX IF NOT EXISTS idx_warehouse_type ON public.warehouse (type_warehouse);
CREATE INDEX IF NOT EXISTS idx_warehouse_plant_id ON public.warehouse (plant_id);

COMMIT;
