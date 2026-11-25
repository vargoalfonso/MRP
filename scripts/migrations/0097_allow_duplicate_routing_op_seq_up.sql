-- Allow multiple routing operations to use the same op_seq within a header.
ALTER TABLE routing_operations
DROP CONSTRAINT IF EXISTS routing_operations_routing_header_id_op_seq_key;