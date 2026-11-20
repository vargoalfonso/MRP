ALTER TABLE IF EXISTS scrap_stocks
    DROP COLUMN IF EXISTS source_defect_id,
    DROP COLUMN IF EXISTS source_qc_log_id;

DROP TABLE IF EXISTS qc_defect_items;
DROP TABLE IF EXISTS qc_logs;
