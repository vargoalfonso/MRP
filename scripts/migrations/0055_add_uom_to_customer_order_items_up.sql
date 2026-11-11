ALTER TABLE customer_order_document_items
    ADD COLUMN IF NOT EXISTS uom VARCHAR(64);
