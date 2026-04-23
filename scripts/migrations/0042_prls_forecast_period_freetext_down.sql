-- Revert forecast_period back to the original strict quarter format.
-- WARNING: This will fail if existing data contains non-matching values.

ALTER TABLE IF EXISTS public.prls
    ALTER COLUMN forecast_period TYPE VARCHAR(7);

ALTER TABLE IF EXISTS public.prls
    DROP CONSTRAINT IF EXISTS prls_forecast_period_check;

ALTER TABLE IF EXISTS public.prls
    ADD CONSTRAINT prls_forecast_period_check CHECK (forecast_period ~ '^[0-9]{4}-Q[1-4]$');
