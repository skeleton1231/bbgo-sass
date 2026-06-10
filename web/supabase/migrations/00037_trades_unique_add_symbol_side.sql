-- =============================================
-- Add symbol and side to trades unique key to match original MySQL (exchange, symbol, side, id)
-- This handles self-trades where the same trade ID appears on both sides,
-- and prevents theoretical conflicts when exchange reuses trade IDs across symbols.
-- =============================================

-- Live trades
DROP INDEX IF EXISTS trades_user_trade_uniq;
CREATE UNIQUE INDEX trades_user_trade_uniq
  ON public.trades(user_id, exchange, symbol, side, trade_id);

-- Paper trades
DROP INDEX IF EXISTS paper_trades_user_trade_uniq;
CREATE UNIQUE INDEX paper_trades_user_trade_uniq
  ON public.paper_trades(user_id, exchange, symbol, side, trade_id);
