-- =========================
-- EXTENSION
-- =========================
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- global_parameters
CREATE TABLE
    IF NOT EXISTS delivery_notes (
        id BIGSERIAL PRIMARY KEY,
        dn_number VARCHAR(50) NOT NULL UNIQUE,
        period VARCHAR(50),
        customer_id BIGINT,
        contact_person VARCHAR(100),
        po_number VARCHAR(50) NOT NULL,
        type VARCHAR(50),
        status VARCHAR(50) DEFAULT 'draft',
        incoming_date DATE,
        created_by VARCHAR(100),
        created_at TIMESTAMPTZ DEFAULT NOW (),
        updated_at TIMESTAMPTZ DEFAULT NOW ()
    );

CREATE TABLE
    IF NOT EXISTS delivery_note_items (
        id BIGSERIAL PRIMARY KEY,
        dn_id BIGINT NOT NULL,
        item_uniq_code VARCHAR(100) NOT NULL,
        quantity INT NOT NULL,
        uom VARCHAR(50),
        weight INT,
        kanban_id BIGINT,
        qr TEXT,
        created_at TIMESTAMPTZ DEFAULT NOW (),
        updated_at TIMESTAMPTZ DEFAULT NOW ()
    );
