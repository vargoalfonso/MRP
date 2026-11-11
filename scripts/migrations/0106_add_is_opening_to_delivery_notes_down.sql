-- Down:
DROP INDEX IF EXISTS idx_delivery_notes_is_opening;
-- Hapus item packing opening lebih dulu agar tidak yatim.
DELETE FROM delivery_note_items
 WHERE dn_id IN (SELECT id FROM delivery_notes WHERE is_opening = TRUE);
DELETE FROM delivery_notes WHERE is_opening = TRUE;
ALTER TABLE delivery_notes DROP COLUMN IF EXISTS is_opening;
