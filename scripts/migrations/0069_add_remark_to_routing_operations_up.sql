-- Add remark column to routing_operations
ALTER TABLE routing_operations
ADD COLUMN IF NOT EXISTS remark text;
