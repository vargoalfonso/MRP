-- Restore the previous uniqueness rule on rollback.

ALTER TABLE public.supplier_item
    DROP CONSTRAINT IF EXISTS supplier_item_supplier_uniq_type_key;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'supplier_item_supplier_uniq_key'
          AND conrelid = 'public.supplier_item'::regclass
    ) THEN
        ALTER TABLE public.supplier_item
            ADD CONSTRAINT supplier_item_supplier_uniq_key
            UNIQUE (supplier_uuid, uniq_code);
    END IF;
END $$;
