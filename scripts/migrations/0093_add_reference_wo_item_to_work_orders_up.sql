-- =============================================================================
-- Migration: 0093_add_reference_wo_item_to_work_orders_up.sql
-- Feature  : WO Rework -> Reason/Info Defect per kanban sumber
-- Notes    : reference_wo_item_id menunjuk ke work_order_items SUMBER (kanban
--            spesifik) yang memicu rework. Dipakai untuk memfilter Reason/Info
--            Defect di detail WO agar tidak menggabungkan semua kanban ber-uniq
--            sama. Runner migrasi hanya menjalankan STATEMENT PERTAMA per file,
--            jadi cukup satu ALTER TABLE.
-- =============================================================================

ALTER TABLE IF EXISTS work_orders
    ADD COLUMN IF NOT EXISTS reference_wo_item_id BIGINT;
