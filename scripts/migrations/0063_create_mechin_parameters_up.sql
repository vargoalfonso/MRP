CREATE TABLE IF NOT EXISTS mechin_parameters (
    id BIGSERIAL PRIMARY KEY,
    machine_name VARCHAR(150) NOT NULL,
    machine_count INTEGER NOT NULL,
    operating_hours INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'Active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_mechin_parameters_status ON mechin_parameters (status);

CREATE INDEX IF NOT EXISTS idx_mechin_parameters_deleted_at ON mechin_parameters (deleted_at);