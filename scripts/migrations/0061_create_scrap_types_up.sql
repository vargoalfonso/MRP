-- Migration: 0061_create_scrap_types_up.sql
-- Create scrap_types table for scrap type master data

CREATE TABLE IF NOT EXISTS scrap_types (
    id              BIGSERIAL PRIMARY KEY,
    code            VARCHAR(32) NOT NULL UNIQUE,
    name            VARCHAR(128) NOT NULL,
    description     TEXT,
    status          VARCHAR(20) NOT NULL DEFAULT 'Active',
    is_system       BOOLEAN NOT NULL DEFAULT FALSE,
    created_by      UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_scrap_types_name ON scrap_types(name) WHERE deleted_at IS NULL;
CREATE INDEX idx_scrap_types_status ON scrap_types(status);

-- Seed default scrap types
INSERT INTO scrap_types (code, name, description, is_system) VALUES
    ('SCR-001', 'Machine Scrap', 'Scrap from machine operation', true),
    ('SCR-002', 'Process Scrap', 'Scrap from production process', true),
    ('SCR-003', 'Product Return Scrap', 'Scrap from customer product return', true);
