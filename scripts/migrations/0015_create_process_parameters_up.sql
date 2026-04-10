-- =========================
-- EXTENSION
-- =========================
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- process_parameters
CREATE TABLE process_parameters (
    id SERIAL PRIMARY KEY,
    process_code VARCHAR(50),
    process_name VARCHAR(100),
    category VARCHAR(50),
    sequence INT,
    status VARCHAR(20),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
