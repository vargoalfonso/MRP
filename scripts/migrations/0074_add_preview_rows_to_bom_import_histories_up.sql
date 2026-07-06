ALTER TABLE bom_import_histories
    ADD COLUMN IF NOT EXISTS preview_rows JSONB;