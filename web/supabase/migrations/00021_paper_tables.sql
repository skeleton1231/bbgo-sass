-- Paper trading tables: mirror live tables with paper_ prefix
-- bbgo paper containers write to these tables when DB_DRIVER=supabase + SUPABASE_TABLE_PREFIX=paper_

-- =============================================
-- paper_orders (mirrors orders)
-- =============================================
CREATE TABLE public.paper_orders (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID NOT NULL REFERENCES public.user_profiles(id) ON DELETE CASCADE,
  symbol TEXT NOT NULL DEFAULT '',
  side TEXT NOT NULL DEFAULT '',
  order_type TEXT NOT NULL DEFAULT '',
  price TEXT NOT NULL DEFAULT '0',
  quantity TEXT NOT NULL DEFAULT '0',
  status TEXT NOT NULL DEFAULT '',
  order_id TEXT NOT NULL DEFAULT '',
  executed_quantity TEXT,
  exchange TEXT NOT NULL DEFAULT '',
  client_order_id TEXT NOT NULL DEFAULT '',
  time_in_force TEXT NOT NULL DEFAULT '',
  stop_price TEXT NOT NULL DEFAULT '0',
  is_working BOOLEAN NOT NULL DEFAULT false,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  is_margin BOOLEAN NOT NULL DEFAULT false,
  is_isolated BOOLEAN NOT NULL DEFAULT false,
  is_futures BOOLEAN NOT NULL DEFAULT false,
  order_uuid TEXT NOT NULL DEFAULT '',
  actual_order_id BIGINT NOT NULL DEFAULT 0,
  strategy_instance_id TEXT NOT NULL DEFAULT ''
);

ALTER TABLE public.paper_orders ENABLE ROW LEVEL SECURITY;

CREATE POLICY "Service role manages paper_orders"
  ON public.paper_orders FOR ALL
  USING (auth.role() = 'service_role');
CREATE POLICY "Users can view own paper orders"
  ON public.paper_orders FOR SELECT
  USING (auth.uid() = user_id);

CREATE UNIQUE INDEX paper_orders_user_order_uniq
  ON public.paper_orders(user_id, order_id);
CREATE INDEX idx_paper_orders_user_symbol
  ON public.paper_orders(user_id, symbol);
CREATE INDEX idx_paper_orders_exchange_symbol
  ON public.paper_orders(user_id, exchange, symbol);
CREATE INDEX idx_paper_orders_strategy_instance
  ON public.paper_orders(user_id, strategy_instance_id);

-- =============================================
-- paper_trades (mirrors trades)
-- =============================================
CREATE TABLE public.paper_trades (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID NOT NULL REFERENCES public.user_profiles(id) ON DELETE CASCADE,
  symbol TEXT NOT NULL DEFAULT '',
  side TEXT NOT NULL DEFAULT '',
  price TEXT NOT NULL DEFAULT '0',
  quantity TEXT NOT NULL DEFAULT '0',
  fee TEXT NOT NULL DEFAULT '0',
  fee_currency TEXT NOT NULL DEFAULT '',
  trade_id TEXT NOT NULL DEFAULT '',
  order_id TEXT NOT NULL DEFAULT '',
  pnl TEXT,
  quote_quantity TEXT,
  traded_at TIMESTAMPTZ,
  exchange TEXT NOT NULL DEFAULT '',
  is_buyer BOOLEAN NOT NULL DEFAULT false,
  is_maker BOOLEAN NOT NULL DEFAULT false,
  is_margin BOOLEAN NOT NULL DEFAULT false,
  is_isolated BOOLEAN NOT NULL DEFAULT false,
  strategy TEXT NOT NULL DEFAULT '',
  is_futures BOOLEAN NOT NULL DEFAULT false,
  order_uuid TEXT NOT NULL DEFAULT '',
  strategy_instance_id TEXT NOT NULL DEFAULT ''
);

ALTER TABLE public.paper_trades ENABLE ROW LEVEL SECURITY;

CREATE POLICY "Service role manages paper_trades"
  ON public.paper_trades FOR ALL
  USING (auth.role() = 'service_role');
CREATE POLICY "Users can view own paper trades"
  ON public.paper_trades FOR SELECT
  USING (auth.uid() = user_id);

CREATE UNIQUE INDEX paper_trades_user_trade_uniq
  ON public.paper_trades(user_id, trade_id);
CREATE INDEX idx_paper_trades_user_symbol
  ON public.paper_trades(user_id, symbol);
CREATE INDEX idx_paper_trades_exchange_symbol
  ON public.paper_trades(user_id, exchange, symbol);
CREATE INDEX idx_paper_trades_traded_at
  ON public.paper_trades(user_id, traded_at DESC);
CREATE INDEX idx_paper_trades_strategy_instance
  ON public.paper_trades(user_id, strategy_instance_id);

-- =============================================
-- paper_positions (mirrors positions)
-- =============================================
CREATE TABLE public.paper_positions (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID NOT NULL REFERENCES public.user_profiles(id) ON DELETE CASCADE,
  strategy TEXT NOT NULL DEFAULT '',
  strategy_instance_id TEXT NOT NULL DEFAULT '',
  symbol TEXT NOT NULL DEFAULT '',
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

ALTER TABLE public.paper_positions ENABLE ROW LEVEL SECURITY;

CREATE POLICY "Service role manages paper_positions"
  ON public.paper_positions FOR ALL
  USING (auth.role() = 'service_role');
CREATE POLICY "Users can view own paper positions"
  ON public.paper_positions FOR SELECT
  USING (auth.uid() = user_id);

CREATE INDEX idx_paper_positions_user_symbol
  ON public.paper_positions(user_id, symbol);
CREATE INDEX idx_paper_positions_strategy_instance
  ON public.paper_positions(user_id, strategy_instance_id);

-- =============================================
-- paper_profits (mirrors profits)
-- =============================================
CREATE TABLE public.paper_profits (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID NOT NULL REFERENCES public.user_profiles(id) ON DELETE CASCADE,
  strategy TEXT NOT NULL DEFAULT '',
  strategy_instance_id TEXT NOT NULL DEFAULT '',
  symbol TEXT NOT NULL DEFAULT '',
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

ALTER TABLE public.paper_profits ENABLE ROW LEVEL SECURITY;

CREATE POLICY "Service role manages paper_profits"
  ON public.paper_profits FOR ALL
  USING (auth.role() = 'service_role');
CREATE POLICY "Users can view own paper profits"
  ON public.paper_profits FOR SELECT
  USING (auth.uid() = user_id);

CREATE INDEX idx_paper_profits_user_symbol
  ON public.paper_profits(user_id, symbol);
CREATE INDEX idx_paper_profits_traded_at_symbol
  ON public.paper_profits(user_id, traded_at DESC, symbol);
CREATE INDEX idx_paper_profits_strategy_instance
  ON public.paper_profits(user_id, strategy_instance_id);

-- =============================================
-- Realtime publication for paper tables
-- =============================================
ALTER PUBLICATION supabase_realtime ADD TABLE public.paper_orders;
ALTER PUBLICATION supabase_realtime ADD TABLE public.paper_trades;
ALTER PUBLICATION supabase_realtime ADD TABLE public.paper_positions;
ALTER PUBLICATION supabase_realtime ADD TABLE public.paper_profits;
