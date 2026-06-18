DROP INDEX IF EXISTS public.idx_customers_bom_codes_gin;

ALTER TABLE public.customers
    DROP CONSTRAINT IF EXISTS customers_bom_codes_array_check;

ALTER TABLE public.customers
    DROP COLUMN IF EXISTS bom_codes;
