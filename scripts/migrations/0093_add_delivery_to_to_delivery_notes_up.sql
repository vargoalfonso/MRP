-- Up: tambah kolom pengiriman ke delivery_notes.
-- delivery_to    = "pengiriman ke ..." (1..N)
-- delivery_total = total pengiriman terjadwal (= cycle_time / lead time supplier)
ALTER TABLE delivery_notes
  ADD COLUMN IF NOT EXISTS delivery_to INT,
  ADD COLUMN IF NOT EXISTS delivery_total INT;
