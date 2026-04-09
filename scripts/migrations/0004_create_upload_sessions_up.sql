-- =============================================================================
-- Migration: 0004_create_upload_sessions_up.sql
-- Module   : Chunked Upload (resumable file upload for 2D/3D CAD assets)
-- =============================================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- =============================================================================
-- upload_sessions  — one session per file upload attempt
-- =============================================================================
CREATE TABLE IF NOT EXISTS upload_sessions (
    id           UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    item_id      BIGINT       NOT NULL REFERENCES items(id),
    -- drawing / photo / 3d-model / other
    asset_type   VARCHAR(32)  NOT NULL CHECK (asset_type IN ('drawing','photo','3d-model','other')),
    file_name    VARCHAR(255) NOT NULL,
    mime_type    VARCHAR(100) NULL,
    file_size    BIGINT       NOT NULL,   -- total file size in bytes
    chunk_size   INT          NOT NULL,   -- bytes per chunk (agreed at session create)
    total_chunks INT          NOT NULL,
    -- pending / uploading / assembling / completed / failed / cancelled
    status       VARCHAR(20)  NOT NULL DEFAULT 'pending'
                              CHECK (status IN ('pending','uploading','assembling','completed','failed','cancelled')),
    final_url    TEXT         NULL,       -- filled after successful assemble
    expires_at   TIMESTAMPTZ  NOT NULL,   -- session TTL (e.g. now + 24h)
    created_by   UUID         NULL,       -- users.uuid
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_upload_sessions_item_id   ON upload_sessions(item_id);
CREATE INDEX IF NOT EXISTS idx_upload_sessions_status    ON upload_sessions(status);
CREATE INDEX IF NOT EXISTS idx_upload_sessions_expires   ON upload_sessions(expires_at);

-- =============================================================================
-- upload_chunks  — one row per received chunk
-- =============================================================================
CREATE TABLE IF NOT EXISTS upload_chunks (
    id           BIGSERIAL    PRIMARY KEY,
    session_id   UUID         NOT NULL REFERENCES upload_sessions(id) ON DELETE CASCADE,
    chunk_index  INT          NOT NULL,   -- 0-based
    file_path    TEXT         NOT NULL,   -- tmp path on disk: tmp/uploads/{session_id}/{index}.chunk
    size         BIGINT       NOT NULL,   -- actual bytes received
    checksum     VARCHAR(64)  NULL,       -- hex MD5 sent by client (optional verify)
    uploaded_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (session_id, chunk_index)
);

CREATE INDEX IF NOT EXISTS idx_upload_chunks_session_id ON upload_chunks(session_id);
