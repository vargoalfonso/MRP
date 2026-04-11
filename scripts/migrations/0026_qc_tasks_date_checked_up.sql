-- Add date_checked to qc_tasks for incoming QC approval.
ALTER TABLE qc_tasks
    ADD COLUMN IF NOT EXISTS date_checked date;
