-- =============================================================================
-- Migration: 0051_customer_order_documents_add_so_down.sql
-- Feature  : Rollback SO support on customer_order_documents
-- =============================================================================

DO $$
BEGIN
    IF to_regclass('public.customer_order_documents') IS NOT NULL THEN
        IF EXISTS (
            SELECT 1
            FROM pg_constraint
            WHERE conname = 'customer_order_documents_type_ck'
        ) THEN
            ALTER TABLE customer_order_documents
                DROP CONSTRAINT customer_order_documents_type_ck;
        END IF;

        ALTER TABLE customer_order_documents
            ADD CONSTRAINT customer_order_documents_type_ck
            CHECK (document_type IN ('PO', 'DN'));
    END IF;
END $$;
