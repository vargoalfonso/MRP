-- customer_order_documents: add contact_person, delivery_address, created_by; drop source_reference; add SO type
ALTER TABLE customer_order_documents
    ADD COLUMN IF NOT EXISTS contact_person  VARCHAR(255),
    ADD COLUMN IF NOT EXISTS delivery_address TEXT,
    ADD COLUMN IF NOT EXISTS created_by      VARCHAR(255),
    DROP COLUMN IF EXISTS source_reference;

ALTER TABLE customer_order_documents
    DROP CONSTRAINT IF EXISTS customer_order_documents_type_ck;

ALTER TABLE customer_order_documents
    ADD CONSTRAINT customer_order_documents_type_ck
        CHECK (document_type IN ('PO', 'DN', 'SO'));

-- customer_order_document_items: drop fields not used in frontend
ALTER TABLE customer_order_document_items
    DROP COLUMN IF EXISTS planning_date,
    DROP COLUMN IF EXISTS make_lead_time_days,
    DROP COLUMN IF EXISTS delivery_cycle,
    DROP COLUMN IF EXISTS uom;
