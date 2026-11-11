-- Add the status column used by the safety stock create/update API.
ALTER TABLE IF EXISTS public.safety_stock_parameters
    ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'Active';
