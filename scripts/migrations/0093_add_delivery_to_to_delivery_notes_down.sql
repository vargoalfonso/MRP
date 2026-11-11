-- Down
ALTER TABLE delivery_notes
  DROP COLUMN IF EXISTS delivery_to,
  DROP COLUMN IF EXISTS delivery_total;
