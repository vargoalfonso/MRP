-- =============================================================================
-- Migration: 0032_create_approval_instances_up.sql
-- Table    : approval_instances
-- Purpose  : Generic approval tracker — satu baris per dokumen yang sedang
--            atau sudah melalui proses approval. Tabel ini bersifat generic:
--            bisa dipakai oleh BOM, PO, DN, dan modul lain tanpa perlu tabel
--            approval terpisah per modul.
--
-- Kolom utama:
--   action_name       → nama modul (bom, po, dn, dll) — dipakai frontend
--                        untuk redirect ke halaman detail modul yang sesuai
--   reference_id      → ID dokumen di modul tersebut (bom_item.id, po.id, dst)
--   approval_workflow_id → FK ke master approval_workflows
--   current_level     → level yang sedang menunggu approve (1..4)
--   max_level         → jumlah level aktif, dihitung saat submit dari workflow
--   status            → pending | approved | rejected
--   approval_progress → JSONB, track per level: role, status, approved_by,
--                        approved_at, note
--
-- Struktur approval_progress JSONB:
-- {
--   "levels": [
--     {
--       "level": 1,
--       "role": "manager_produksi",
--       "status": "approved",          -- pending | approved | rejected | skipped
--       "approved_by": "uuid-user",
--       "approved_at": "2026-04-13T10:00:00Z",
--       "note": "Sudah sesuai spek"
--     },
--     { "level": 2, "role": "kepala_teknik", "status": "pending", ... },
--     { "level": 3, "role": null, "status": "skipped", ... },
--     { "level": 4, "role": null, "status": "skipped", ... }
--   ]
-- }
--
-- Level yang tidak dikonfigurasi di approval_workflows akan diisi "skipped"
-- saat record pertama kali dibuat (saat user submit dokumen).
-- =============================================================================

CREATE TABLE IF NOT EXISTS approval_instances (
    id                   BIGINT       GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

action_name VARCHAR(100) NOT NULL,


reference_table VARCHAR(100) NOT NULL,


reference_id     BIGINT       NOT NULL,

approval_workflow_id BIGINT NOT NULL REFERENCES approval_workflows (id) ON DELETE RESTRICT,

current_level INT NOT NULL DEFAULT 1 CHECK (current_level BETWEEN 1 AND 4),


max_level INT NOT NULL CHECK (max_level BETWEEN 2 AND 4),

status VARCHAR(20) NOT NULL DEFAULT 'pending',

submitted_by VARCHAR(100) NULL,



approval_progress    JSONB        NOT NULL DEFAULT '{}',

    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Index untuk query by modul + dokumen
CREATE INDEX IF NOT EXISTS idx_approval_instances_action_ref ON approval_instances (
    action_name,
    reference_table,
    reference_id
);

-- Index untuk query semua approval yang sedang pending di suatu workflow
CREATE INDEX IF NOT EXISTS idx_approval_instances_workflow_status ON approval_instances (approval_workflow_id, status);

-- Index JSONB untuk query status per level
CREATE INDEX IF NOT EXISTS idx_approval_instances_progress ON approval_instances USING GIN (approval_progress);