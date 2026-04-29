-- Migration: 0062_create_machine_patterns_up.sql
-- Create machine_patterns table for machine pattern master data

CREATE TABLE IF NOT EXISTS machine_patterns (
    id                  BIGSERIAL PRIMARY KEY,
    uniq_code           VARCHAR(64) NOT NULL,
    machine_id          BIGINT NOT NULL REFERENCES master_machines(id),
    cycle_time          DECIMAL(10,2),
    pattern_value       DECIMAL(5,2) NOT NULL DEFAULT 1.0,
    working_days        INTEGER NOT NULL DEFAULT 26,
    moving_type         VARCHAR(20) NOT NULL DEFAULT 'Normal',
    min_output          DECIMAL(10,2) DEFAULT 0,
    prl_reference       DECIMAL(15,4),
    status              VARCHAR(20) NOT NULL DEFAULT 'Active',
    created_by          UUID,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ
);

CREATE INDEX idx_machine_patterns_uniq ON machine_patterns(uniq_code);
CREATE INDEX idx_machine_patterns_machine ON machine_patterns(machine_id);
CREATE UNIQUE INDEX idx_machine_patterns_uniq_machine ON machine_patterns(uniq_code, machine_id) WHERE deleted_at IS NULL;
