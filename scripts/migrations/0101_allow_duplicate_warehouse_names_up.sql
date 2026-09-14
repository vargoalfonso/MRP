-- Warehouse names may repeat, including within the same plant.
-- UUID remains unique so each warehouse record is still addressable.
BEGIN;

ALTER TABLE IF EXISTS public.warehouse
    DROP CONSTRAINT IF EXISTS warehouse_name_plant_key;

-- In case an older environment created a standalone unique index instead of
-- the named constraint, remove that index too after the constraint is gone.
DROP INDEX IF EXISTS public.warehouse_name_plant_key;

COMMIT;
