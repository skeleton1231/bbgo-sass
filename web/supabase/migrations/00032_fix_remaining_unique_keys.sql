-- =============================================
-- Fix remaining unique keys to match original bbgo MySQL schema
--   trades:  MySQL (exchange, symbol, side, id)
--   profits: MySQL (exchange, symbol, side, trade_id)
--   positions: MySQL latest (trade_id, side, symbol, exchange)
-- =============================================

-- trades
DROP INDEX IF EXISTS trades_user_trade_uniq;
CREATE UNIQUE INDEX trades_user_trade_uniq
  ON public.trades(user_id, exchange, trade_id);

-- paper_trades
DROP INDEX IF EXISTS paper_trades_user_trade_uniq;
CREATE UNIQUE INDEX paper_trades_user_trade_uniq
  ON public.paper_trades(user_id, exchange, trade_id);

-- profits
DROP INDEX IF EXISTS profits_user_trade_uniq;
CREATE UNIQUE INDEX profits_user_trade_uniq
  ON public.profits(user_id, exchange, symbol, side, trade_id);

-- paper_profits
DROP INDEX IF EXISTS paper_profits_user_trade_uniq;
CREATE UNIQUE INDEX paper_profits_user_trade_uniq
  ON public.paper_profits(user_id, exchange, symbol, side, trade_id);

-- positions
DROP INDEX IF EXISTS positions_user_trade_side_exchange_uniq;
CREATE UNIQUE INDEX positions_user_trade_side_exchange_uniq
  ON public.positions(user_id, trade_id, side, symbol, exchange);

-- paper_positions
DROP INDEX IF EXISTS paper_positions_user_trade_side_exchange_uniq;
CREATE UNIQUE INDEX paper_positions_user_trade_side_exchange_uniq
  ON public.paper_positions(user_id, trade_id, side, symbol, exchange);
