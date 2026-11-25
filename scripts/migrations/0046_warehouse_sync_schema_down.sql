-- Down migration for 0046.
-- WARNING: This attempts to remove the columns/constraints added by 0046.
-- Only run this if you fully understand the impact.

BEGIN;

ALTER TABLE IF EXISTS public.warehouse
    DROP CONSTRAINT IF EXISTS warehouse_type_check,
    DROP CONSTRAINT IF EXISTS warehouse_name_plant_key,
    DROP CONSTRAINT IF EXISTS warehouse_uuid_key;

DROP INDEX IF EXISTS public.idx_warehouse_type;
DROP INDEX IF EXISTS public.idx_warehouse_plant_id;
-- Note: idx_warehouse_deleted_at may be used by other migrations.

ALTER TABLE IF EXISTS public.warehouse
    DROP COLUMN IF EXISTS uuid,
    DROP COLUMN IF EXISTS warehouse_name,
    DROP COLUMN IF EXISTS type_warehouse,
    DROP COLUMN IF EXISTS plant_id,
    DROP COLUMN IF EXISTS created_at,
    DROP COLUMN IF EXISTS updated_at;

COMMIT;
