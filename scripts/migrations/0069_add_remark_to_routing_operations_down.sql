-- Rollback for 0069_add_remark_to_routing_operations_up.sql
ALTER TABLE routing_operations
DROP COLUMN IF EXISTS remark;
