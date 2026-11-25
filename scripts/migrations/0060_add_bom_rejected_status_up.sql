-- Add 'Rejected' to allowed bom_item status values
-- This allows BOM approval rejection to set status to 'Rejected'

ALTER TABLE bom_item DROP CONSTRAINT IF EXISTS bom_item_status_check;
ALTER TABLE bom_item ADD CONSTRAINT bom_item_status_check CHECK (status IN ('Draft', 'Released', 'Obsolete', 'Rejected'));
