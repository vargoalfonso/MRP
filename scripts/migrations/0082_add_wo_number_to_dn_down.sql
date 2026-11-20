-- Down: remove wo_number from delivery_notes and delivery_note_suppliers
DROP INDEX IF EXISTS idx_delivery_note_suppliers_wo_number;
DROP INDEX IF EXISTS idx_delivery_notes_wo_number;

ALTER TABLE delivery_note_suppliers
  DROP COLUMN IF EXISTS wo_number;

ALTER TABLE delivery_notes
  DROP COLUMN IF EXISTS wo_number;
