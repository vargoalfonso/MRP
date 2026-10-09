-- Up: tandai delivery_notes sintetis "OPENING-STOCK".
-- DN ini dibuat otomatis untuk menampung Initial Packing dari opening stock
-- raw material (stok awal yang di-inject tanpa melalui DN supplier), supaya
-- stok tersebut punya Packing ID/QR dan bisa di-scan produksi.
-- supplier_id sengaja dibiarkan NULL agar tidak ikut supplier performance.
ALTER TABLE delivery_notes
  ADD COLUMN IF NOT EXISTS is_opening BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_delivery_notes_is_opening
  ON delivery_notes (is_opening) WHERE is_opening = TRUE;
