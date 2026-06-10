-- =============================================
-- Paper futures improvements:
-- 1. strategy_instance_id on futures_position_risks for multi-instance isolation
-- 2. strategy_instance_id on paper_balances for consistency
-- =============================================

-- paper_futures_position_risks: add strategy_instance_id for multi-instance isolation
ALTER TABLE public.paper_futures_position_risks
  ADD COLUMN IF NOT EXISTS strategy_instance_id TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_paper_fpr_strategy_instance
  ON public.paper_futures_position_risks(user_id, strategy_instance_id);

-- paper_balances: add strategy_instance_id for consistency
ALTER TABLE public.paper_balances
  ADD COLUMN IF NOT EXISTS strategy_instance_id TEXT NOT NULL DEFAULT '';
