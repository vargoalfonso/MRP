-- +migrate Up
-- Make sebango_code and uniq_code nullable TEXT and add material_code
ALTER TABLE supplier_item
    ALTER COLUMN sebango_code DROP NOT NULL,
    ALTER COLUMN sebango_code TYPE TEXT;

ALTER TABLE supplier_item
    ALTER COLUMN uniq_code DROP NOT NULL,
    ALTER COLUMN uniq_code TYPE TEXT;

ALTER TABLE supplier_item
    ADD COLUMN IF NOT EXISTS material_code VARCHAR(100);

-- +migrate Down
-- Revert changes: make sebango_code and uniq_code varchar(100) NOT NULL and drop material_code
ALTER TABLE supplier_item
    DROP COLUMN IF EXISTS material_code;

-- Only set columns back to NOT NULL when no NULL values exist; otherwise leave as-is and raise NOTICE
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM supplier_item WHERE sebango_code IS NULL) THEN
        ALTER TABLE supplier_item
            ALTER COLUMN sebango_code TYPE VARCHAR(100),
            ALTER COLUMN sebango_code SET NOT NULL;
    ELSE
        RAISE NOTICE 'sebango_code contains NULL values; cannot set NOT NULL in down migration';
    END IF;

    IF NOT EXISTS (SELECT 1 FROM supplier_item WHERE uniq_code IS NULL) THEN
        ALTER TABLE supplier_item
            ALTER COLUMN uniq_code TYPE VARCHAR(100),
            ALTER COLUMN uniq_code SET NOT NULL;
    ELSE
        RAISE NOTICE 'uniq_code contains NULL values; cannot set NOT NULL in down migration';
    END IF;
END$$;
