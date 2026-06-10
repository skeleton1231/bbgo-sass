-- =============================================
-- Change futures_position_risks from upsert to append (match MySQL behavior)
-- Drop unique constraints so each sync inserts a new row, preserving history.
-- =============================================

-- Live table: drop unique, keep regular index for queries
DROP INDEX IF EXISTS futures_position_risks_user_symbol_side_instance_uniq;
DROP INDEX IF EXISTS futures_position_risks_user_symbol_side_uniq;
CREATE INDEX IF NOT EXISTS idx_fpr_user_symbol
  ON public.futures_position_risks(user_id, exchange, symbol);

-- Paper table: drop unique, keep regular index for queries
DROP INDEX IF EXISTS paper_futures_position_risks_user_symbol_side_uniq;
CREATE INDEX IF NOT EXISTS idx_paper_fpr_user_symbol
  ON public.paper_futures_position_risks(user_id, exchange, symbol);
