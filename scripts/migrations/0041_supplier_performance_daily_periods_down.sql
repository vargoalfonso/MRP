ALTER TABLE public.supplier_performance_snapshots
    DROP CONSTRAINT IF EXISTS supplier_performance_period_type_check;

ALTER TABLE public.supplier_performance_snapshots
    ADD CONSTRAINT supplier_performance_period_type_check
        CHECK (evaluation_period_type IN ('monthly', 'quarterly', 'yearly'));
