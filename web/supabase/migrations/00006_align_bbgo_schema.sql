-- =============================================
-- Align Supabase schema with bbgo original design
-- - Rename sync_orders → orders, sync_trades → trades
-- - Drop extra columns not in bbgo original
-- - Rename type → order_type in orders
-- - Drop legacy bots table
-- =============================================

-- 1. Drop bot_id foreign keys from both tables
ALTER TABLE public.sync_orders DROP COLUMN IF EXISTS bot_id;
ALTER TABLE public.sync_trades DROP COLUMN IF EXISTS bot_id;

-- 2. Drop extra columns not in bbgo original design
-- sync_orders: drop synced_at, creation_time (bbgo uses created_at)
ALTER TABLE public.sync_orders DROP COLUMN IF EXISTS synced_at;
ALTER TABLE public.sync_orders DROP COLUMN IF EXISTS creation_time;

-- sync_trades: drop synced_at, created_at (bbgo uses traded_at)
ALTER TABLE public.sync_trades DROP COLUMN IF EXISTS synced_at;
ALTER TABLE public.sync_trades DROP COLUMN IF EXISTS created_at;

-- 3. Rename type → order_type in sync_orders (matches bbgo original)
ALTER TABLE public.sync_orders RENAME COLUMN type TO order_type;

-- 4. Drop old indexes (will be recreated with new table names)
DROP INDEX IF EXISTS sync_orders_user_order_uniq;
DROP INDEX IF EXISTS idx_sync_orders_user_symbol;
DROP INDEX IF EXISTS idx_sync_orders_exchange_symbol;
DROP INDEX IF EXISTS sync_trades_user_trade_uniq;
DROP INDEX IF EXISTS idx_sync_trades_user_symbol;
DROP INDEX IF EXISTS idx_sync_trades_exchange_symbol;
DROP INDEX IF EXISTS idx_sync_trades_traded_at;

-- 5. Rename tables to match bbgo original
ALTER TABLE public.sync_orders RENAME TO orders;
ALTER TABLE public.sync_trades RENAME TO trades;

-- 6. Recreate indexes with new table names
CREATE UNIQUE INDEX IF NOT EXISTS orders_user_order_uniq
  ON public.orders(user_id, order_id);
CREATE INDEX IF NOT EXISTS idx_orders_user_symbol
  ON public.orders(user_id, symbol);
CREATE INDEX IF NOT EXISTS idx_orders_exchange_symbol
  ON public.orders(user_id, exchange, symbol);

CREATE UNIQUE INDEX IF NOT EXISTS trades_user_trade_uniq
  ON public.trades(user_id, trade_id);
CREATE INDEX IF NOT EXISTS idx_trades_user_symbol
  ON public.trades(user_id, symbol);
CREATE INDEX IF NOT EXISTS idx_trades_exchange_symbol
  ON public.trades(user_id, exchange, symbol);
CREATE INDEX IF NOT EXISTS idx_trades_traded_at
  ON public.trades(user_id, traded_at DESC);

-- 7. Drop bots table (legacy, strategies live in user_containers.strategies JSONB)
DROP TRIGGER IF EXISTS set_updated_at ON public.bots;
DROP TABLE IF EXISTS public.bots;
