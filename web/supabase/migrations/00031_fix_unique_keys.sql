-- =============================================
-- Fix unique keys to include `exchange` column
-- Matches original bbgo MySQL schema design:
--   orders:                (order_id, exchange)
--   margin_liquidations:   (order_id, exchange)
--   futures_position_risks:(exchange, symbol, position_side)
-- =============================================

-- Drop old indexes and recreate with exchange in the key

-- orders
DROP INDEX IF EXISTS orders_user_order_uniq;
CREATE UNIQUE INDEX orders_user_order_uniq
  ON public.orders(user_id, order_id, exchange);

-- paper_orders
DROP INDEX IF EXISTS paper_orders_user_order_uniq;
CREATE UNIQUE INDEX paper_orders_user_order_uniq
  ON public.paper_orders(user_id, order_id, exchange);

-- margin_liquidations
DROP INDEX IF EXISTS margin_liquidations_user_order_uniq;
CREATE UNIQUE INDEX margin_liquidations_user_order_uniq
  ON public.margin_liquidations(user_id, order_id, exchange);

-- paper_margin_liquidations
DROP INDEX IF EXISTS paper_margin_liquidations_user_order_uniq;
CREATE UNIQUE INDEX paper_margin_liquidations_user_order_uniq
  ON public.paper_margin_liquidations(user_id, order_id, exchange);

-- futures_position_risks
DROP INDEX IF EXISTS futures_position_risks_user_symbol_side_uniq;
CREATE UNIQUE INDEX futures_position_risks_user_symbol_side_uniq
  ON public.futures_position_risks(user_id, exchange, symbol, position_side);

-- paper_futures_position_risks
DROP INDEX IF EXISTS paper_futures_position_risks_user_symbol_side_uniq;
CREATE UNIQUE INDEX paper_futures_position_risks_user_symbol_side_uniq
  ON public.paper_futures_position_risks(user_id, exchange, symbol, position_side);
