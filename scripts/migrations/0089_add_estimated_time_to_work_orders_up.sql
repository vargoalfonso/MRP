-- Migration: 0089_add_estimated_time_to_work_orders_up.sql
-- [wo-estimated-time] Simpan estimasi waktu produksi WO (menit).
-- Rumus: estimated_time_minutes = SUM(qty_item x cycle_time_min x machine_capacity)
-- cycle_time_min & machine_capacity disimpan sebagai snapshot supaya angka
-- estimasi tetap bisa ditelusuri walaupun master BOM/mesin berubah.

ALTER TABLE work_orders
    ADD COLUMN IF NOT EXISTS estimated_time_minutes NUMERIC(15,4),
    ADD COLUMN IF NOT EXISTS cycle_time_min         NUMERIC(15,4),
    ADD COLUMN IF NOT EXISTS machine_capacity       NUMERIC(15,4);

COMMENT ON COLUMN work_orders.estimated_time_minutes IS 'Estimasi total waktu produksi WO dalam menit';
COMMENT ON COLUMN work_orders.cycle_time_min IS 'Snapshot cycle time (menit/pcs) dari BOM saat WO dibuat';
COMMENT ON COLUMN work_orders.machine_capacity IS 'Snapshot machine capacity mesin yang dipakai saat WO dibuat';
