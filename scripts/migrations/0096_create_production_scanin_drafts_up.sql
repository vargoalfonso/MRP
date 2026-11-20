-- Migration: 0096_create_production_scanin_drafts_up.sql
-- [scanin-draft-db] Persist draft scan-in (seed) di DB agar semua gadget tim
-- produksi melihat hasil scan yang sama tanpa perlu scan ulang. Menggantikan
-- penyimpanan localStorage per-browser (erp_frontliner_production_scanout_seed).
--
-- Satu baris = draft satu kanban (wo_item_id) pada satu step proses.
-- payload berisi ProductionScanOutSeed (mesin, qty, alokasi packing/RM).

CREATE TABLE IF NOT EXISTS production_scanin_drafts (
    id            BIGSERIAL PRIMARY KEY,
    wo_id         BIGINT NOT NULL,
    wo_item_id    BIGINT NOT NULL,
    current_step  INTEGER NOT NULL DEFAULT 1,
    payload       JSONB NOT NULL,
    updated_by    VARCHAR(255),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_production_scanin_drafts_key UNIQUE (wo_id, wo_item_id, current_step)
);

CREATE INDEX IF NOT EXISTS idx_production_scanin_drafts_wo
    ON production_scanin_drafts (wo_id, current_step);
