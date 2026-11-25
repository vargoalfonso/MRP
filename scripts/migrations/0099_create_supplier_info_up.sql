CREATE TABLE IF NOT EXISTS supplier_info (
    id             BIGSERIAL PRIMARY KEY,
    uuid           VARCHAR(36) NOT NULL UNIQUE,
    uniq           VARCHAR(255) NOT NULL,
    uniq_zahir     VARCHAR(255),
    supplier_name  VARCHAR(255) NOT NULL,
    type           VARCHAR(50)  NOT NULL,
    status         VARCHAR(20)  NOT NULL DEFAULT 'active',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at     TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_supplier_info_uniq ON supplier_info(uniq);
CREATE INDEX IF NOT EXISTS idx_supplier_info_deleted_at ON supplier_info(deleted_at);
