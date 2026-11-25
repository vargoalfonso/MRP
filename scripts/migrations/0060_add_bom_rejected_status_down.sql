-- Revert: Remove 'Rejected' from allowed bom_item status values

ALTER TABLE bom_item DROP CONSTRAINT IF EXISTS bom_item_status_check;
ALTER TABLE bom_item ADD CONSTRAINT bom_item_status_check CHECK (status IN ('Draft', 'Released', 'Obsolete'));
