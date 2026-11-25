-- Revert: restore dropped columns, remove added columns
ALTER TABLE customer_order_document_items
    ADD COLUMN IF NOT EXISTS uom               VARCHAR(32) NOT NULL DEFAULT 'pcs',
    ADD COLUMN IF NOT EXISTS delivery_cycle    VARCHAR(100),
    ADD COLUMN IF NOT EXISTS make_lead_time_days INTEGER NOT NULL DEFAULT 0 CHECK (make_lead_time_days >= 0);

ALTER TABLE customer_order_documents
    DROP CONSTRAINT IF EXISTS customer_order_documents_type_ck;

ALTER TABLE customer_order_documents
    ADD CONSTRAINT customer_order_documents_type_ck
        CHECK (document_type IN ('PO', 'DN'));

ALTER TABLE customer_order_documents
    DROP COLUMN IF EXISTS contact_person,
    DROP COLUMN IF EXISTS delivery_address,
    DROP COLUMN IF EXISTS created_by,
    ADD COLUMN IF NOT EXISTS source_reference VARCHAR(128);
