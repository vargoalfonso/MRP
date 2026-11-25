DROP TABLE IF EXISTS public.customers;
DROP TABLE IF EXISTS public.suppliers;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public' AND table_name = 'suppliers_legacy_0006'
    ) THEN
        ALTER TABLE public.suppliers_legacy_0006 RENAME TO suppliers;
    END IF;
END $$;
