-- [so-packing] rollback
DROP INDEX IF EXISTS idx_so_entries_packing_number;

ALTER TABLE stock_opname_entries
    DROP COLUMN IF EXISTS packing_number,
    DROP COLUMN IF EXISTS dn_number,
    DROP COLUMN IF EXISTS max_qty;
