-- =========================
-- EXTENSION
-- =========================
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- global_parameters
CREATE TABLE IF NOT EXISTS kanban_parameters (
    id BIGSERIAL PRIMARY KEY,
    item_uniq_code VARCHAR(100) NOT NULL,
    kanban_qty INT NOT NULL,
    min_stock INT DEFAULT 0,
    max_stock INT DEFAULT 0,
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

