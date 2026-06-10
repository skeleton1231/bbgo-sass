-- =============================================
-- Drop inline UNIQUE constraints from CREATE TABLE
-- that coexist with the named indexes created in 00032
-- Inline constraints have system-generated names (*_key)
-- =============================================

-- positions: inline UNIQUE(user_id, trade_id, side, exchange) replaced by named index (user_id, trade_id, side, symbol, exchange)
ALTER TABLE public.positions DROP CONSTRAINT IF EXISTS positions_user_id_trade_id_side_exchange_key;
ALTER TABLE public.paper_positions DROP CONSTRAINT IF EXISTS paper_positions_user_id_trade_id_side_exchange_key;

-- profits: inline UNIQUE(user_id, trade_id) replaced by named index (user_id, exchange, symbol, side, trade_id)
ALTER TABLE public.profits DROP CONSTRAINT IF EXISTS profits_user_id_trade_id_key;
ALTER TABLE public.paper_profits DROP CONSTRAINT IF EXISTS paper_profits_user_id_trade_id_key;
