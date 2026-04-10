-- =========================
-- EXTENSION
-- =========================
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- approval_workflows
CREATE TABLE approval_workflows (
    id SERIAL PRIMARY KEY,
    action_name VARCHAR(100),
    level_1_role VARCHAR(50),
    level_2_role VARCHAR(50),
    level_3_role VARCHAR(50),
    level_4_role VARCHAR(50),
    status VARCHAR(20),
    created_by VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
