-- [so-packing] Stock opname di Action UI di-scan per packing list / DN.
-- packing_number : nomor packing list (delivery_note_items.packing_number)
-- dn_number      : DN pemilik packing tsb (delivery_notes.dn_number)
-- max_qty        : qty maksimal packing saat entry dibuat (batas counted_qty)
ALTER TABLE stock_opname_entries
    ADD COLUMN IF NOT EXISTS packing_number VARCHAR(100),
    ADD COLUMN IF NOT EXISTS dn_number      VARCHAR(100),
    ADD COLUMN IF NOT EXISTS max_qty        NUMERIC(15,4);

CREATE INDEX IF NOT EXISTS idx_so_entries_packing_number
    ON stock_opname_entries (packing_number);
