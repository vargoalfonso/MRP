-- Add the status column used by the safety stock create/update API.
ALTER TABLE IF EXISTS public.safety_stock_parameters
    ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'Active';

UPDATE public.safety_stock_parameters
SET status = 'Active'
WHERE status IS NULL;

ALTER TABLE public.safety_stock_parameters
    ALTER COLUMN status SET DEFAULT 'Active',
    ALTER COLUMN status SET NOT NULL;
