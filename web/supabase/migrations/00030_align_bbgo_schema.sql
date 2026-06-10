-- =============================================
-- Align Supabase schema with original bbgo MySQL design
-- - Add `total` column to paper_balances (bbgo SQLite: currency, total, available, locked)
-- - Add strategy_instance_id indexes to live tables (matching paper tables)
-- - Add gid column to orders/trades for bbgo struct scan compatibility
-- =============================================

-- =============================================
-- paper_balances: add `total` column to match bbgo SQLite schema
-- bbgo SQLite INSERT: (currency, total, available, locked)
-- bbgo Balance.Total() = Available + Locked
-- =============================================
ALTER TABLE paper_balances ADD COLUMN IF NOT EXISTS total TEXT NOT NULL DEFAULT '0';

-- =============================================
-- Live tables: add strategy_instance_id indexes (paper tables already have these)
-- =============================================
CREATE INDEX IF NOT EXISTS idx_orders_strategy_instance
  ON public.orders(user_id, strategy_instance_id);
CREATE INDEX IF NOT EXISTS idx_trades_strategy_instance
  ON public.trades(user_id, strategy_instance_id);

-- =============================================
-- Add gid column to live tables for struct scan compatibility
-- bbgo Order struct has `GID uint64 db:"gid"` and Trade has `GID int64 db:"gid"`
-- These are used by StructScan in query functions
-- =============================================
ALTER TABLE public.orders ADD COLUMN IF NOT EXISTS gid BIGSERIAL;
ALTER TABLE public.trades ADD COLUMN IF NOT EXISTS gid BIGSERIAL;

-- =============================================
-- Paper tables: add gid column too
-- =============================================
ALTER TABLE public.paper_orders ADD COLUMN IF NOT EXISTS gid BIGSERIAL;
ALTER TABLE public.paper_trades ADD COLUMN IF NOT EXISTS gid BIGSERIAL;
