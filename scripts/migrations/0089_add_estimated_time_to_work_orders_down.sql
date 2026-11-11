-- Migration: 0089_add_estimated_time_to_work_orders_down.sql

ALTER TABLE work_orders
    DROP COLUMN IF EXISTS estimated_time_minutes,
    DROP COLUMN IF EXISTS cycle_time_min,
    DROP COLUMN IF EXISTS machine_capacity;
