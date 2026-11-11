-- +migrate Down
-- Hanya total_po_incoming yang dibuat migrasi ini dengan pasti; qty_sent dan "check"
-- mungkin sudah ada sebelumnya, jadi tidak dihapus.
ALTER TABLE delivery_notes
  DROP COLUMN IF EXISTS total_po_incoming;
