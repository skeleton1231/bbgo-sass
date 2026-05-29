-- =============================================
-- Phase 5: Full bbgo direct-write schema
-- Consolidates all tables/columns from manager/migrate.go
-- so migrate.go can be removed entirely.
-- =============================================

-- =============================================
-- Expand sync_orders with full bbgo columns
-- =============================================
ALTER TABLE public.sync_orders ADD COLUMN IF NOT EXISTS exchange TEXT NOT NULL DEFAULT '';
ALTER TABLE public.sync_orders ADD COLUMN IF NOT EXISTS client_order_id TEXT NOT NULL DEFAULT '';
ALTER TABLE public.sync_orders ADD COLUMN IF NOT EXISTS time_in_force TEXT NOT NULL DEFAULT '';
ALTER TABLE public.sync_orders ADD COLUMN IF NOT EXISTS stop_price TEXT NOT NULL DEFAULT '0';
ALTER TABLE public.sync_orders ADD COLUMN IF NOT EXISTS is_working BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE public.sync_orders ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE public.sync_orders ADD COLUMN IF NOT EXISTS is_margin BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE public.sync_orders ADD COLUMN IF NOT EXISTS is_isolated BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE public.sync_orders ADD COLUMN IF NOT EXISTS is_futures BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE public.sync_orders ADD COLUMN IF NOT EXISTS order_uuid TEXT NOT NULL DEFAULT '';
ALTER TABLE public.sync_orders ADD COLUMN IF NOT EXISTS actual_order_id BIGINT NOT NULL DEFAULT 0;

-- Unique index for bbgo upsert (ON CONFLICT)
CREATE UNIQUE INDEX IF NOT EXISTS sync_orders_user_order_uniq
  ON public.sync_orders(user_id, order_id);

CREATE INDEX IF NOT EXISTS idx_sync_orders_user_symbol
  ON public.sync_orders(user_id, symbol);
CREATE INDEX IF NOT EXISTS idx_sync_orders_exchange_symbol
  ON public.sync_orders(user_id, exchange, symbol);

-- =============================================
-- Expand sync_trades with full bbgo columns
-- =============================================
ALTER TABLE public.sync_trades ADD COLUMN IF NOT EXISTS exchange TEXT NOT NULL DEFAULT '';
ALTER TABLE public.sync_trades ADD COLUMN IF NOT EXISTS is_buyer BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE public.sync_trades ADD COLUMN IF NOT EXISTS is_maker BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE public.sync_trades ADD COLUMN IF NOT EXISTS is_margin BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE public.sync_trades ADD COLUMN IF NOT EXISTS is_isolated BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE public.sync_trades ADD COLUMN IF NOT EXISTS strategy TEXT NOT NULL DEFAULT '';
ALTER TABLE public.sync_trades ADD COLUMN IF NOT EXISTS is_futures BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE public.sync_trades ADD COLUMN IF NOT EXISTS order_uuid TEXT NOT NULL DEFAULT '';

-- Unique index for bbgo upsert (ON CONFLICT)
CREATE UNIQUE INDEX IF NOT EXISTS sync_trades_user_trade_uniq
  ON public.sync_trades(user_id, trade_id);

CREATE INDEX IF NOT EXISTS idx_sync_trades_user_symbol
  ON public.sync_trades(user_id, symbol);
CREATE INDEX IF NOT EXISTS idx_sync_trades_exchange_symbol
  ON public.sync_trades(user_id, exchange, symbol);
CREATE INDEX IF NOT EXISTS idx_sync_trades_traded_at
  ON public.sync_trades(user_id, traded_at DESC);

-- =============================================
-- Positions table (bbgo position tracking)
-- =============================================
CREATE TABLE IF NOT EXISTS public.positions (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID NOT NULL REFERENCES public.user_profiles(id) ON DELETE CASCADE,
  strategy TEXT NOT NULL,
  strategy_instance_id TEXT NOT NULL DEFAULT '',
  symbol TEXT NOT NULL,
  quote_currency TEXT NOT NULL DEFAULT '',
  base_currency TEXT NOT NULL DEFAULT '',
  average_cost TEXT NOT NULL DEFAULT '0',
  base TEXT NOT NULL DEFAULT '0',
  quote TEXT NOT NULL DEFAULT '0',
  profit TEXT,
  net_profit TEXT,
  trade_id BIGINT NOT NULL,
  side TEXT NOT NULL DEFAULT '',
  exchange TEXT NOT NULL DEFAULT '',
  traded_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(user_id, trade_id, side, exchange)
);

ALTER TABLE public.positions ENABLE ROW LEVEL SECURITY;

DO $$ BEGIN
  CREATE POLICY "Service role manages positions"
    ON public.positions FOR ALL
    USING (auth.role() = 'service_role');
  CREATE POLICY "Users can view own positions"
    ON public.positions FOR SELECT
    USING (auth.uid() = user_id);
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE INDEX IF NOT EXISTS idx_positions_user_symbol
  ON public.positions(user_id, symbol);

-- =============================================
-- Profits table (bbgo profit tracking)
-- =============================================
CREATE TABLE IF NOT EXISTS public.profits (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID NOT NULL REFERENCES public.user_profiles(id) ON DELETE CASCADE,
  strategy TEXT NOT NULL,
  strategy_instance_id TEXT NOT NULL DEFAULT '',
  symbol TEXT NOT NULL,
  average_cost TEXT NOT NULL DEFAULT '0',
  profit TEXT NOT NULL DEFAULT '0',
  net_profit TEXT NOT NULL DEFAULT '0',
  profit_margin TEXT NOT NULL DEFAULT '0',
  net_profit_margin TEXT NOT NULL DEFAULT '0',
  quote_currency TEXT NOT NULL DEFAULT '',
  base_currency TEXT NOT NULL DEFAULT '',
  exchange TEXT NOT NULL DEFAULT '',
  is_futures BOOLEAN NOT NULL DEFAULT false,
  is_margin BOOLEAN NOT NULL DEFAULT false,
  is_isolated BOOLEAN NOT NULL DEFAULT false,
  trade_id BIGINT NOT NULL,
  side TEXT NOT NULL DEFAULT '',
  is_buyer BOOLEAN NOT NULL DEFAULT false,
  is_maker BOOLEAN NOT NULL DEFAULT false,
  price TEXT NOT NULL DEFAULT '0',
  quantity TEXT NOT NULL DEFAULT '0',
  quote_quantity TEXT NOT NULL DEFAULT '0',
  traded_at TIMESTAMPTZ NOT NULL,
  fee_in_usd TEXT,
  fee TEXT NOT NULL DEFAULT '0',
  fee_currency TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(user_id, trade_id)
);

ALTER TABLE public.profits ENABLE ROW LEVEL SECURITY;

DO $$ BEGIN
  CREATE POLICY "Service role manages profits"
    ON public.profits FOR ALL
    USING (auth.role() = 'service_role');
  CREATE POLICY "Users can view own profits"
    ON public.profits FOR SELECT
    USING (auth.uid() = user_id);
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE INDEX IF NOT EXISTS idx_profits_user_symbol
  ON public.profits(user_id, symbol);
CREATE INDEX IF NOT EXISTS idx_profits_traded_at_symbol
  ON public.profits(user_id, traded_at DESC, symbol);

-- =============================================
-- Drop sync_cursors (no longer needed with direct write)
-- =============================================
DROP TABLE IF EXISTS public.sync_cursors;
