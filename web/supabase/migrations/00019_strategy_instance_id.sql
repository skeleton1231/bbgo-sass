-- Add strategy_instance_id to trades and orders for per-strategy-instance data isolation
ALTER TABLE trades ADD COLUMN IF NOT EXISTS strategy_instance_id text NOT NULL DEFAULT '';
ALTER TABLE orders ADD COLUMN IF NOT EXISTS strategy_instance_id text NOT NULL DEFAULT '';
