CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS customer_order_documents (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT uuid_generate_v4() UNIQUE,
    document_type VARCHAR(16) NOT NULL,
    document_number VARCHAR(128) NOT NULL,
    document_date DATE NOT NULL,
    period_schedule VARCHAR(64),
    customer_id BIGINT NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    customer_name_snapshot VARCHAR(255),
    status VARCHAR(32) NOT NULL DEFAULT 'draft',
    source_reference VARCHAR(128),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT customer_order_documents_type_ck
        CHECK (document_type IN ('PO', 'DN')),
    CONSTRAINT customer_order_documents_status_ck
        CHECK (status IN ('draft', 'active', 'completed', 'cancelled')),
    CONSTRAINT customer_order_documents_type_number_uq
        UNIQUE (document_type, document_number)
);

CREATE INDEX IF NOT EXISTS idx_customer_order_documents_customer_id
    ON customer_order_documents (customer_id);

CREATE INDEX IF NOT EXISTS idx_customer_order_documents_type
    ON customer_order_documents (document_type);

CREATE INDEX IF NOT EXISTS idx_customer_order_documents_date
    ON customer_order_documents (document_date);

CREATE INDEX IF NOT EXISTS idx_customer_order_documents_status
    ON customer_order_documents (status);

CREATE INDEX IF NOT EXISTS idx_customer_order_documents_deleted_at
    ON customer_order_documents (deleted_at);

CREATE TABLE IF NOT EXISTS customer_order_document_items (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT uuid_generate_v4() UNIQUE,
    document_id BIGINT NOT NULL REFERENCES customer_order_documents(id) ON DELETE CASCADE,
    line_no INTEGER NOT NULL,
    item_uniq_code VARCHAR(100) NOT NULL,
    model VARCHAR(255),
    part_name VARCHAR(255) NOT NULL,
    part_number VARCHAR(128) NOT NULL,
    quantity NUMERIC(15,4) NOT NULL CHECK (quantity > 0),
    uom VARCHAR(32) NOT NULL,
    delivery_date DATE,
    delivery_cycle VARCHAR(100),
    make_lead_time_days INTEGER NOT NULL DEFAULT 0 CHECK (make_lead_time_days >= 0),
    planning_date DATE GENERATED ALWAYS AS (
        CASE
            WHEN delivery_date IS NULL THEN NULL
            ELSE delivery_date - make_lead_time_days
        END
    ) STORED,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT customer_order_document_items_document_line_uq
        UNIQUE (document_id, line_no)
);

CREATE INDEX IF NOT EXISTS idx_customer_order_document_items_document_id
    ON customer_order_document_items (document_id);

CREATE INDEX IF NOT EXISTS idx_customer_order_document_items_item_uniq_code
    ON customer_order_document_items (item_uniq_code);

CREATE INDEX IF NOT EXISTS idx_customer_order_document_items_part_number
    ON customer_order_document_items (part_number);

CREATE INDEX IF NOT EXISTS idx_customer_order_document_items_delivery_date
    ON customer_order_document_items (delivery_date);

CREATE INDEX IF NOT EXISTS idx_customer_order_document_items_planning_date
    ON customer_order_document_items (planning_date);
