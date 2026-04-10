-- =========================
-- EXTENSION
-- =========================
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- =========================
-- TABLE: unit_measurement
-- =========================
CREATE TABLE IF NOT EXISTS po_split_settings (
    id BIGSERIAL PRIMARY KEY,
    budget_type VARCHAR(50) NOT NULL,
    min_order_qty INT NOT NULL,
    max_split_lines INT NOT NULL,
    split_rule VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
