-- Restore the original uniqueness rule for routing operation sequence numbers.
ALTER TABLE routing_operations
ADD CONSTRAINT routing_operations_routing_header_id_op_seq_key
UNIQUE (routing_header_id, op_seq);