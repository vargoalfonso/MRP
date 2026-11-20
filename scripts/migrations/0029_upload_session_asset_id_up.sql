-- Add optional asset_id to upload_sessions so Complete can update an existing asset (edit flow).
ALTER TABLE upload_sessions
    ADD COLUMN IF NOT EXISTS asset_id bigint REFERENCES item_assets(id) ON DELETE SET NULL;
