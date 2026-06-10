-- =============================================
-- Paper futures improvements:
-- 1. position_action on trades and positions for direction tagging
-- 2. strategy_instance_id on futures_position_risks for multi-instance isolation
-- =============================================

-- paper_trades: add position_action (openLong, closeShort, etc.)
ALTER TABLE public.paper_trades
  ADD COLUMN IF NOT EXISTS position_action TEXT NOT NULL DEFAULT '';

-- paper_positions: add position_action
ALTER TABLE public.paper_positions
  ADD COLUMN IF NOT EXISTS position_action TEXT NOT NULL DEFAULT '';

-- paper_futures_position_risks: add strategy_instance_id for multi-instance isolation
ALTER TABLE public.paper_futures_position_risks
  ADD COLUMN IF NOT EXISTS strategy_instance_id TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_paper_fpr_strategy_instance
  ON public.paper_futures_position_risks(user_id, strategy_instance_id);

-- paper_balances: add strategy_instance_id for consistency
ALTER TABLE public.paper_balances
  ADD COLUMN IF NOT EXISTS strategy_instance_id TEXT NOT NULL DEFAULT '';
