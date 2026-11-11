-- Up: add material_grade column to delivery_notes
ALTER TABLE delivery_notes
  ADD COLUMN IF NOT EXISTS material_grade VARCHAR(100) NULL;

CREATE INDEX IF NOT EXISTS idx_delivery_notes_material_grade ON delivery_notes(material_grade);

-- Down:
-- ALTER TABLE delivery_notes DROP COLUMN IF EXISTS material_grade;
-- DROP INDEX IF EXISTS idx_delivery_notes_material_grade;
