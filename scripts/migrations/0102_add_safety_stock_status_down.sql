-- Roll back the safety stock status column.
ALTER TABLE IF EXISTS public.safety_stock_parameters
    DROP COLUMN IF EXISTS status;
