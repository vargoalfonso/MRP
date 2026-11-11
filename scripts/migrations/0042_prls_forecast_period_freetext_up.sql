-- Allow free-text forecast_period in PRL.
-- 1) Drop the strict format CHECK constraint.
-- 2) Allow unbounded free-text.

ALTER TABLE IF EXISTS public.prls
    DROP CONSTRAINT IF EXISTS prls_forecast_period_check;

ALTER TABLE IF EXISTS public.prls
    ALTER COLUMN forecast_period TYPE TEXT;
