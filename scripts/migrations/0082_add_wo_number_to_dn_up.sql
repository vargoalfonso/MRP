-- Up: add wo_number to delivery_notes and delivery_note_suppliers
-- Supports DN Subcon (surat jalan subcon) linked to a Work Order, used by
-- action-ui DN Subcon OUT / IN scanning to route returned goods to WIP or FG.
ALTER TABLE delivery_notes
  ADD COLUMN IF NOT EXISTS wo_number VARCHAR(64) NULL;

ALTER TABLE delivery_note_suppliers
  ADD COLUMN IF NOT EXISTS wo_number VARCHAR(64) NULL;

CREATE INDEX IF NOT EXISTS idx_delivery_notes_wo_number ON delivery_notes(wo_number);
CREATE INDEX IF NOT EXISTS idx_delivery_note_suppliers_wo_number ON delivery_note_suppliers(wo_number);

-- Down:
-- DROP INDEX IF EXISTS idx_delivery_note_suppliers_wo_number;
-- DROP INDEX IF EXISTS idx_delivery_notes_wo_number;
-- ALTER TABLE delivery_note_suppliers DROP COLUMN IF EXISTS wo_number;
-- ALTER TABLE delivery_notes DROP COLUMN IF EXISTS wo_number;
