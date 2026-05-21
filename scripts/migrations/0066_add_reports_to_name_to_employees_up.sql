-- Add reports_to_name to employees for display convenience
ALTER TABLE employees
  ADD COLUMN IF NOT EXISTS reports_to_name VARCHAR(150);
