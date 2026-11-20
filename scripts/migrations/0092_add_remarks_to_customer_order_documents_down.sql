-- Revert remarks column from customer_order_documents
ALTER TABLE customer_order_documents
DROP COLUMN IF EXISTS remarks;
