-- Restore the previous warehouse_name + plant_id uniqueness rule.
BEGIN;

ALTER TABLE IF EXISTS public.warehouse
    ADD CONSTRAINT warehouse_name_plant_key UNIQUE (warehouse_name, plant_id);

COMMIT;
