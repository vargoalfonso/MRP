-- Add remarks column to customer_order_documents
ALTER TABLE customer_order_documents
ADD COLUMN IF NOT EXISTS remarks TEXT;
