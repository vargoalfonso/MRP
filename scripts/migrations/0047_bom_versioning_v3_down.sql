DROP INDEX IF EXISTS uq_bom_item_one_current_per_item;
DROP INDEX IF EXISTS idx_bom_lines_child_item_revision_id;
DROP INDEX IF EXISTS idx_bom_item_is_current;
DROP INDEX IF EXISTS idx_bom_item_copied_from_bom_id;
DROP INDEX IF EXISTS idx_bom_item_root_item_revision_id;

ALTER TABLE bom_lines
    DROP COLUMN IF EXISTS child_item_revision_id;

ALTER TABLE bom_item
    DROP COLUMN IF EXISTS is_current,
    DROP COLUMN IF EXISTS created_by,
    DROP COLUMN IF EXISTS change_note,
    DROP COLUMN IF EXISTS copied_from_bom_id,
    DROP COLUMN IF EXISTS root_item_revision_id;
