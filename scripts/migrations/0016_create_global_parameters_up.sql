-- =========================
-- EXTENSION
-- =========================
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- global_parameters
CREATE TABLE global_parameters (
    id SERIAL PRIMARY KEY,
    parameter_group VARCHAR(100),
    period VARCHAR(20),
    working_days INT,
    status VARCHAR(20),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

