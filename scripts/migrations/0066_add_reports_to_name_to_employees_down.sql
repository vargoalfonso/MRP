-- Revert: drop reports_to_name from employees
ALTER TABLE employees
  DROP COLUMN IF EXISTS reports_to_name;
