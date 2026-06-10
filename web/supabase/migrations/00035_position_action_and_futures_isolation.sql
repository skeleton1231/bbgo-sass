-- =============================================
-- Add strategy_instance_id to live futures_position_risks
-- for per-instance isolation matching paper tables.
-- =============================================

-- Live futures_position_risks: add strategy_instance_id
ALTER TABLE public.futures_position_risks
  ADD COLUMN IF NOT EXISTS strategy_instance_id TEXT NOT NULL DEFAULT '';

-- Index for per-instance queries on live futures_position_risks
CREATE INDEX IF NOT EXISTS idx_fpr_strategy_instance
  ON public.futures_position_risks(user_id, strategy_instance_id);
