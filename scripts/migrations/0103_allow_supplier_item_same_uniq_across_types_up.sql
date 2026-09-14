-- Allow one supplier to register the same UNIQ in multiple categories.
-- The category is supplier_item.type (raw_material, indirect, subcon).

ALTER TABLE public.supplier_item
    DROP CONSTRAINT IF EXISTS supplier_item_supplier_uniq_key;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'supplier_item_supplier_uniq_type_key'
          AND conrelid = 'public.supplier_item'::regclass
    ) THEN
        ALTER TABLE public.supplier_item
            ADD CONSTRAINT supplier_item_supplier_uniq_type_key
            UNIQUE (supplier_uuid, uniq_code, type);
    END IF;
END $$;
