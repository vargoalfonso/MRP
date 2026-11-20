ALTER TABLE public.customers
    ADD COLUMN IF NOT EXISTS bom_codes JSONB NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE public.customers
    DROP CONSTRAINT IF EXISTS customers_bom_codes_array_check;

ALTER TABLE public.customers
    ADD CONSTRAINT customers_bom_codes_array_check
    CHECK (jsonb_typeof(bom_codes) = 'array');

CREATE INDEX IF NOT EXISTS idx_customers_bom_codes_gin
    ON public.customers USING GIN (bom_codes);
