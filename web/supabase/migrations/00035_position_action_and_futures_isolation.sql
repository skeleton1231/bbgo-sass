-- =============================================
-- Add position_action to orders/trades (live + paper)
-- Add strategy_instance_id to futures_position_risks (live)
-- These enable database-level association of direction (open/close long/short)
-- and per-instance isolation for futures position risks.
-- =============================================

-- Live orders: add position_action (open_long, open_short, close_long, close_short)
ALTER TABLE public.orders
  ADD COLUMN IF NOT EXISTS position_action TEXT NOT NULL DEFAULT '';

-- Live trades: add position_action
ALTER TABLE public.trades
  ADD COLUMN IF NOT EXISTS position_action TEXT NOT NULL DEFAULT '';

-- Live positions: add position_action
ALTER TABLE public.positions
  ADD COLUMN IF NOT EXISTS position_action TEXT NOT NULL DEFAULT '';

-- Paper orders: add position_action (paper_trades and paper_positions already have it from 00034)
ALTER TABLE public.paper_orders
  ADD COLUMN IF NOT EXISTS position_action TEXT NOT NULL DEFAULT '';

-- Live futures_position_risks: add strategy_instance_id (paper table has it from 00034)
ALTER TABLE public.futures_position_risks
  ADD COLUMN IF NOT EXISTS strategy_instance_id TEXT NOT NULL DEFAULT '';

-- Index for per-instance queries on live futures_position_risks
CREATE INDEX IF NOT EXISTS idx_fpr_strategy_instance
  ON public.futures_position_risks(user_id, strategy_instance_id);

-- Drop old unique constraint and recreate with strategy_instance_id
-- so multiple strategies can hold positions on the same symbol
DROP INDEX IF EXISTS futures_position_risks_user_symbol_side_uniq;
CREATE UNIQUE INDEX futures_position_risks_user_symbol_side_instance_uniq
  ON public.futures_position_risks(user_id, exchange, symbol, position_side, strategy_instance_id);
