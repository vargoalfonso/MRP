ALTER TABLE bom_item
    ADD COLUMN IF NOT EXISTS root_item_revision_id BIGINT NULL REFERENCES item_revisions(id),
    ADD COLUMN IF NOT EXISTS copied_from_bom_id BIGINT NULL REFERENCES bom_item(id),
    ADD COLUMN IF NOT EXISTS change_note TEXT NULL,
    ADD COLUMN IF NOT EXISTS created_by UUID NULL,
    ADD COLUMN IF NOT EXISTS is_current BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE bom_lines
    ADD COLUMN IF NOT EXISTS child_item_revision_id BIGINT NULL REFERENCES item_revisions(id);

UPDATE bom_item
SET root_item_revision_id = rev.id
FROM (
    SELECT DISTINCT ON (item_id) item_id, id
    FROM item_revisions
    ORDER BY item_id, id DESC
) rev
WHERE bom_item.item_id = rev.item_id
  AND bom_item.root_item_revision_id IS NULL;

UPDATE bom_item b
SET is_current = TRUE
WHERE b.id IN (
    SELECT DISTINCT ON (item_id) id
    FROM bom_item
    ORDER BY item_id, version DESC, id DESC
);

CREATE INDEX IF NOT EXISTS idx_bom_item_root_item_revision_id ON bom_item(root_item_revision_id);
CREATE INDEX IF NOT EXISTS idx_bom_item_copied_from_bom_id ON bom_item(copied_from_bom_id);
CREATE INDEX IF NOT EXISTS idx_bom_item_is_current ON bom_item(item_id, is_current);
CREATE INDEX IF NOT EXISTS idx_bom_lines_child_item_revision_id ON bom_lines(child_item_revision_id);

CREATE UNIQUE INDEX IF NOT EXISTS uq_bom_item_one_current_per_item
    ON bom_item(item_id)
    WHERE is_current = TRUE;
