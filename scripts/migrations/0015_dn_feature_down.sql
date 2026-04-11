-- +migrate Down

DROP TABLE IF EXISTS incoming_receiving_scans;

DROP TABLE IF EXISTS qc_tasks;

ALTER TABLE IF EXISTS delivery_note_items
DROP COLUMN IF EXISTS order_qty,
DROP COLUMN IF EXISTS date_incoming,
DROP COLUMN IF EXISTS qty_stated,
DROP COLUMN IF EXISTS qty_received,
DROP COLUMN IF EXISTS weight_received,
DROP COLUMN IF EXISTS quality_status,
DROP COLUMN IF EXISTS packing_number,
DROP COLUMN IF EXISTS pcs_per_kanban,
DROP COLUMN IF EXISTS received_at;

ALTER TABLE IF EXISTS delivery_notes
DROP COLUMN IF EXISTS supplier_id,
DROP COLUMN IF EXISTS total_po_qty,
DROP COLUMN IF EXISTS total_dn_created,
DROP COLUMN IF EXISTS total_dn_incoming;