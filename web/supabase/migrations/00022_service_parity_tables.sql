-- =============================================
-- Service parity: all tables needed for full bbgo service alignment
-- Between SQL mode (10 services) and Supabase mode
-- Live + Paper mirrors for each table
-- =============================================

-- =============================================
-- 1. nav_history_details (AccountService)
-- =============================================
CREATE TABLE public.nav_history_details (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID NOT NULL REFERENCES public.user_profiles(id) ON DELETE CASCADE,
  session TEXT NOT NULL DEFAULT '',
  exchange TEXT NOT NULL DEFAULT '',
  subaccount TEXT NOT NULL DEFAULT '',
  time TIMESTAMPTZ NOT NULL DEFAULT now(),
  currency TEXT NOT NULL DEFAULT '',
  net_asset_in_usd TEXT NOT NULL DEFAULT '0',
  net_asset_in_btc TEXT NOT NULL DEFAULT '0',
  balance TEXT NOT NULL DEFAULT '0',
  available TEXT NOT NULL DEFAULT '0',
  locked TEXT NOT NULL DEFAULT '0',
  borrowed TEXT NOT NULL DEFAULT '0',
  net_asset TEXT NOT NULL DEFAULT '0',
  price_in_usd TEXT NOT NULL DEFAULT '0',
  interest TEXT NOT NULL DEFAULT '0',
  is_margin BOOLEAN NOT NULL DEFAULT false,
  is_isolated BOOLEAN NOT NULL DEFAULT false,
  isolated_symbol TEXT NOT NULL DEFAULT ''
);

ALTER TABLE public.nav_history_details ENABLE ROW LEVEL SECURITY;
DO $$ BEGIN
  CREATE POLICY "Service role manages nav_history_details"
    ON public.nav_history_details FOR ALL
    USING (auth.role() = 'service_role');
  CREATE POLICY "Users can view own nav_history_details"
    ON public.nav_history_details FOR SELECT
    USING (auth.uid() = user_id);
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE INDEX idx_nav_history_details_user_time
  ON public.nav_history_details(user_id, time DESC);
CREATE INDEX idx_nav_history_details_user_currency
  ON public.nav_history_details(user_id, currency);

-- =============================================
-- 2. rewards (RewardService)
-- =============================================
CREATE TABLE public.rewards (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID NOT NULL REFERENCES public.user_profiles(id) ON DELETE CASCADE,
  exchange TEXT NOT NULL DEFAULT '',
  uuid TEXT NOT NULL DEFAULT '',
  reward_type TEXT NOT NULL DEFAULT '',
  currency TEXT NOT NULL DEFAULT '',
  quantity TEXT NOT NULL DEFAULT '0',
  state TEXT NOT NULL DEFAULT '',
  note TEXT,
  spent BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE public.rewards ENABLE ROW LEVEL SECURITY;
DO $$ BEGIN
  CREATE POLICY "Service role manages rewards"
    ON public.rewards FOR ALL
    USING (auth.role() = 'service_role');
  CREATE POLICY "Users can view own rewards"
    ON public.rewards FOR SELECT
    USING (auth.uid() = user_id);
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE UNIQUE INDEX rewards_user_uuid_uniq
  ON public.rewards(user_id, uuid);
CREATE INDEX idx_rewards_user_exchange
  ON public.rewards(user_id, exchange);

-- =============================================
-- 3. withdraws (WithdrawService)
-- =============================================
CREATE TABLE public.withdraws (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID NOT NULL REFERENCES public.user_profiles(id) ON DELETE CASCADE,
  exchange TEXT NOT NULL DEFAULT '',
  asset TEXT NOT NULL DEFAULT '',
  network TEXT NOT NULL DEFAULT '',
  address TEXT NOT NULL DEFAULT '',
  amount TEXT NOT NULL DEFAULT '0',
  txn_id TEXT NOT NULL DEFAULT '',
  txn_fee TEXT NOT NULL DEFAULT '0',
  txn_fee_currency TEXT NOT NULL DEFAULT '',
  time TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE public.withdraws ENABLE ROW LEVEL SECURITY;
DO $$ BEGIN
  CREATE POLICY "Service role manages withdraws"
    ON public.withdraws FOR ALL
    USING (auth.role() = 'service_role');
  CREATE POLICY "Users can view own withdraws"
    ON public.withdraws FOR SELECT
    USING (auth.uid() = user_id);
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE UNIQUE INDEX withdraws_user_txn_uniq
  ON public.withdraws(user_id, txn_id);
CREATE INDEX idx_withdraws_user_exchange
  ON public.withdraws(user_id, exchange);

-- =============================================
-- 4. deposits (DepositService)
-- =============================================
CREATE TABLE public.deposits (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID NOT NULL REFERENCES public.user_profiles(id) ON DELETE CASCADE,
  exchange TEXT NOT NULL DEFAULT '',
  asset TEXT NOT NULL DEFAULT '',
  address TEXT NOT NULL DEFAULT '',
  amount TEXT NOT NULL DEFAULT '0',
  txn_id TEXT NOT NULL DEFAULT '',
  time TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE public.deposits ENABLE ROW LEVEL SECURITY;
DO $$ BEGIN
  CREATE POLICY "Service role manages deposits"
    ON public.deposits FOR ALL
    USING (auth.role() = 'service_role');
  CREATE POLICY "Users can view own deposits"
    ON public.deposits FOR SELECT
    USING (auth.uid() = user_id);
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE UNIQUE INDEX deposits_user_txn_uniq
  ON public.deposits(user_id, txn_id);
CREATE INDEX idx_deposits_user_exchange
  ON public.deposits(user_id, exchange);

-- =============================================
-- 5. margin_loans (MarginService)
-- =============================================
CREATE TABLE public.margin_loans (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID NOT NULL REFERENCES public.user_profiles(id) ON DELETE CASCADE,
  exchange TEXT NOT NULL DEFAULT '',
  transaction_id BIGINT NOT NULL DEFAULT 0,
  asset TEXT NOT NULL DEFAULT '',
  isolated_symbol TEXT NOT NULL DEFAULT '',
  principle TEXT NOT NULL DEFAULT '0',
  time TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE public.margin_loans ENABLE ROW LEVEL SECURITY;
DO $$ BEGIN
  CREATE POLICY "Service role manages margin_loans"
    ON public.margin_loans FOR ALL
    USING (auth.role() = 'service_role');
  CREATE POLICY "Users can view own margin_loans"
    ON public.margin_loans FOR SELECT
    USING (auth.uid() = user_id);
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE UNIQUE INDEX margin_loans_user_txn_uniq
  ON public.margin_loans(user_id, transaction_id);
CREATE INDEX idx_margin_loans_user_asset
  ON public.margin_loans(user_id, asset);

-- =============================================
-- 6. margin_repays (MarginService)
-- =============================================
CREATE TABLE public.margin_repays (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID NOT NULL REFERENCES public.user_profiles(id) ON DELETE CASCADE,
  exchange TEXT NOT NULL DEFAULT '',
  transaction_id BIGINT NOT NULL DEFAULT 0,
  asset TEXT NOT NULL DEFAULT '',
  isolated_symbol TEXT NOT NULL DEFAULT '',
  principle TEXT NOT NULL DEFAULT '0',
  time TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE public.margin_repays ENABLE ROW LEVEL SECURITY;
DO $$ BEGIN
  CREATE POLICY "Service role manages margin_repays"
    ON public.margin_repays FOR ALL
    USING (auth.role() = 'service_role');
  CREATE POLICY "Users can view own margin_repays"
    ON public.margin_repays FOR SELECT
    USING (auth.uid() = user_id);
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE UNIQUE INDEX margin_repays_user_txn_uniq
  ON public.margin_repays(user_id, transaction_id);
CREATE INDEX idx_margin_repays_user_asset
  ON public.margin_repays(user_id, asset);

-- =============================================
-- 7. margin_interests (MarginService)
-- =============================================
CREATE TABLE public.margin_interests (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID NOT NULL REFERENCES public.user_profiles(id) ON DELETE CASCADE,
  exchange TEXT NOT NULL DEFAULT '',
  asset TEXT NOT NULL DEFAULT '',
  isolated_symbol TEXT NOT NULL DEFAULT '',
  principle TEXT NOT NULL DEFAULT '0',
  interest TEXT NOT NULL DEFAULT '0',
  interest_rate TEXT NOT NULL DEFAULT '0',
  time TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE public.margin_interests ENABLE ROW LEVEL SECURITY;
DO $$ BEGIN
  CREATE POLICY "Service role manages margin_interests"
    ON public.margin_interests FOR ALL
    USING (auth.role() = 'service_role');
  CREATE POLICY "Users can view own margin_interests"
    ON public.margin_interests FOR SELECT
    USING (auth.uid() = user_id);
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE INDEX idx_margin_interests_user_asset
  ON public.margin_interests(user_id, asset);
CREATE INDEX idx_margin_interests_user_time
  ON public.margin_interests(user_id, time DESC);

-- =============================================
-- 8. margin_liquidations (MarginService)
-- =============================================
CREATE TABLE public.margin_liquidations (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID NOT NULL REFERENCES public.user_profiles(id) ON DELETE CASCADE,
  exchange TEXT NOT NULL DEFAULT '',
  symbol TEXT NOT NULL DEFAULT '',
  side TEXT NOT NULL DEFAULT '',
  order_id BIGINT NOT NULL DEFAULT 0,
  price TEXT NOT NULL DEFAULT '0',
  quantity TEXT NOT NULL DEFAULT '0',
  average_price TEXT NOT NULL DEFAULT '0',
  executed_quantity TEXT NOT NULL DEFAULT '0',
  time_in_force TEXT NOT NULL DEFAULT '',
  is_isolated BOOLEAN NOT NULL DEFAULT false,
  time TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE public.margin_liquidations ENABLE ROW LEVEL SECURITY;
DO $$ BEGIN
  CREATE POLICY "Service role manages margin_liquidations"
    ON public.margin_liquidations FOR ALL
    USING (auth.role() = 'service_role');
  CREATE POLICY "Users can view own margin_liquidations"
    ON public.margin_liquidations FOR SELECT
    USING (auth.uid() = user_id);
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE UNIQUE INDEX margin_liquidations_user_order_uniq
  ON public.margin_liquidations(user_id, order_id);
CREATE INDEX idx_margin_liquidations_user_symbol
  ON public.margin_liquidations(user_id, symbol);

-- =============================================
-- 9. futures_position_risks (FuturesService)
-- =============================================
CREATE TABLE public.futures_position_risks (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID NOT NULL REFERENCES public.user_profiles(id) ON DELETE CASCADE,
  exchange TEXT NOT NULL DEFAULT '',
  symbol TEXT NOT NULL DEFAULT '',
  position_side TEXT NOT NULL DEFAULT '',
  leverage TEXT NOT NULL DEFAULT '0',
  liquidation_price TEXT NOT NULL DEFAULT '0',
  entry_price TEXT NOT NULL DEFAULT '0',
  mark_price TEXT NOT NULL DEFAULT '0',
  break_even_price TEXT NOT NULL DEFAULT '0',
  position_amount TEXT NOT NULL DEFAULT '0',
  unrealized_pnl TEXT NOT NULL DEFAULT '0',
  notional TEXT NOT NULL DEFAULT '0',
  initial_margin TEXT NOT NULL DEFAULT '0',
  maint_margin TEXT NOT NULL DEFAULT '0',
  position_initial_margin TEXT NOT NULL DEFAULT '0',
  open_order_initial_margin TEXT NOT NULL DEFAULT '0',
  adl TEXT NOT NULL DEFAULT '0',
  margin_asset TEXT NOT NULL DEFAULT '',
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE public.futures_position_risks ENABLE ROW LEVEL SECURITY;
DO $$ BEGIN
  CREATE POLICY "Service role manages futures_position_risks"
    ON public.futures_position_risks FOR ALL
    USING (auth.role() = 'service_role');
  CREATE POLICY "Users can view own futures_position_risks"
    ON public.futures_position_risks FOR SELECT
    USING (auth.uid() = user_id);
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE UNIQUE INDEX futures_position_risks_user_symbol_side_uniq
  ON public.futures_position_risks(user_id, symbol, position_side);
CREATE INDEX idx_futures_position_risks_user_exchange
  ON public.futures_position_risks(user_id, exchange);

-- =============================================
-- Paper mirrors (identical structure, paper_ prefix)
-- =============================================

-- paper_nav_history_details
CREATE TABLE public.paper_nav_history_details (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID NOT NULL REFERENCES public.user_profiles(id) ON DELETE CASCADE,
  session TEXT NOT NULL DEFAULT '',
  exchange TEXT NOT NULL DEFAULT '',
  subaccount TEXT NOT NULL DEFAULT '',
  time TIMESTAMPTZ NOT NULL DEFAULT now(),
  currency TEXT NOT NULL DEFAULT '',
  net_asset_in_usd TEXT NOT NULL DEFAULT '0',
  net_asset_in_btc TEXT NOT NULL DEFAULT '0',
  balance TEXT NOT NULL DEFAULT '0',
  available TEXT NOT NULL DEFAULT '0',
  locked TEXT NOT NULL DEFAULT '0',
  borrowed TEXT NOT NULL DEFAULT '0',
  net_asset TEXT NOT NULL DEFAULT '0',
  price_in_usd TEXT NOT NULL DEFAULT '0',
  interest TEXT NOT NULL DEFAULT '0',
  is_margin BOOLEAN NOT NULL DEFAULT false,
  is_isolated BOOLEAN NOT NULL DEFAULT false,
  isolated_symbol TEXT NOT NULL DEFAULT ''
);

ALTER TABLE public.paper_nav_history_details ENABLE ROW LEVEL SECURITY;
CREATE POLICY "Service role manages paper_nav_history_details"
  ON public.paper_nav_history_details FOR ALL
  USING (auth.role() = 'service_role');
CREATE POLICY "Users can view own paper_nav_history_details"
  ON public.paper_nav_history_details FOR SELECT
  USING (auth.uid() = user_id);

CREATE INDEX idx_paper_nav_history_details_user_time
  ON public.paper_nav_history_details(user_id, time DESC);

-- paper_rewards
CREATE TABLE public.paper_rewards (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID NOT NULL REFERENCES public.user_profiles(id) ON DELETE CASCADE,
  exchange TEXT NOT NULL DEFAULT '',
  uuid TEXT NOT NULL DEFAULT '',
  reward_type TEXT NOT NULL DEFAULT '',
  currency TEXT NOT NULL DEFAULT '',
  quantity TEXT NOT NULL DEFAULT '0',
  state TEXT NOT NULL DEFAULT '',
  note TEXT,
  spent BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE public.paper_rewards ENABLE ROW LEVEL SECURITY;
CREATE POLICY "Service role manages paper_rewards"
  ON public.paper_rewards FOR ALL
  USING (auth.role() = 'service_role');
CREATE POLICY "Users can view own paper_rewards"
  ON public.paper_rewards FOR SELECT
  USING (auth.uid() = user_id);

CREATE UNIQUE INDEX paper_rewards_user_uuid_uniq
  ON public.paper_rewards(user_id, uuid);

-- paper_withdraws
CREATE TABLE public.paper_withdraws (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID NOT NULL REFERENCES public.user_profiles(id) ON DELETE CASCADE,
  exchange TEXT NOT NULL DEFAULT '',
  asset TEXT NOT NULL DEFAULT '',
  network TEXT NOT NULL DEFAULT '',
  address TEXT NOT NULL DEFAULT '',
  amount TEXT NOT NULL DEFAULT '0',
  txn_id TEXT NOT NULL DEFAULT '',
  txn_fee TEXT NOT NULL DEFAULT '0',
  txn_fee_currency TEXT NOT NULL DEFAULT '',
  time TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE public.paper_withdraws ENABLE ROW LEVEL SECURITY;
CREATE POLICY "Service role manages paper_withdraws"
  ON public.paper_withdraws FOR ALL
  USING (auth.role() = 'service_role');
CREATE POLICY "Users can view own paper_withdraws"
  ON public.paper_withdraws FOR SELECT
  USING (auth.uid() = user_id);

CREATE UNIQUE INDEX paper_withdraws_user_txn_uniq
  ON public.paper_withdraws(user_id, txn_id);

-- paper_deposits
CREATE TABLE public.paper_deposits (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID NOT NULL REFERENCES public.user_profiles(id) ON DELETE CASCADE,
  exchange TEXT NOT NULL DEFAULT '',
  asset TEXT NOT NULL DEFAULT '',
  address TEXT NOT NULL DEFAULT '',
  amount TEXT NOT NULL DEFAULT '0',
  txn_id TEXT NOT NULL DEFAULT '',
  time TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE public.paper_deposits ENABLE ROW LEVEL SECURITY;
CREATE POLICY "Service role manages paper_deposits"
  ON public.paper_deposits FOR ALL
  USING (auth.role() = 'service_role');
CREATE POLICY "Users can view own paper_deposits"
  ON public.paper_deposits FOR SELECT
  USING (auth.uid() = user_id);

CREATE UNIQUE INDEX paper_deposits_user_txn_uniq
  ON public.paper_deposits(user_id, txn_id);

-- paper_margin_loans
CREATE TABLE public.paper_margin_loans (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID NOT NULL REFERENCES public.user_profiles(id) ON DELETE CASCADE,
  exchange TEXT NOT NULL DEFAULT '',
  transaction_id BIGINT NOT NULL DEFAULT 0,
  asset TEXT NOT NULL DEFAULT '',
  isolated_symbol TEXT NOT NULL DEFAULT '',
  principle TEXT NOT NULL DEFAULT '0',
  time TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE public.paper_margin_loans ENABLE ROW LEVEL SECURITY;
CREATE POLICY "Service role manages paper_margin_loans"
  ON public.paper_margin_loans FOR ALL
  USING (auth.role() = 'service_role');
CREATE POLICY "Users can view own paper_margin_loans"
  ON public.paper_margin_loans FOR SELECT
  USING (auth.uid() = user_id);

CREATE UNIQUE INDEX paper_margin_loans_user_txn_uniq
  ON public.paper_margin_loans(user_id, transaction_id);

-- paper_margin_repays
CREATE TABLE public.paper_margin_repays (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID NOT NULL REFERENCES public.user_profiles(id) ON DELETE CASCADE,
  exchange TEXT NOT NULL DEFAULT '',
  transaction_id BIGINT NOT NULL DEFAULT 0,
  asset TEXT NOT NULL DEFAULT '',
  isolated_symbol TEXT NOT NULL DEFAULT '',
  principle TEXT NOT NULL DEFAULT '0',
  time TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE public.paper_margin_repays ENABLE ROW LEVEL SECURITY;
CREATE POLICY "Service role manages paper_margin_repays"
  ON public.paper_margin_repays FOR ALL
  USING (auth.role() = 'service_role');
CREATE POLICY "Users can view own paper_margin_repays"
  ON public.paper_margin_repays FOR SELECT
  USING (auth.uid() = user_id);

CREATE UNIQUE INDEX paper_margin_repays_user_txn_uniq
  ON public.paper_margin_repays(user_id, transaction_id);

-- paper_margin_interests
CREATE TABLE public.paper_margin_interests (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID NOT NULL REFERENCES public.user_profiles(id) ON DELETE CASCADE,
  exchange TEXT NOT NULL DEFAULT '',
  asset TEXT NOT NULL DEFAULT '',
  isolated_symbol TEXT NOT NULL DEFAULT '',
  principle TEXT NOT NULL DEFAULT '0',
  interest TEXT NOT NULL DEFAULT '0',
  interest_rate TEXT NOT NULL DEFAULT '0',
  time TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE public.paper_margin_interests ENABLE ROW LEVEL SECURITY;
CREATE POLICY "Service role manages paper_margin_interests"
  ON public.paper_margin_interests FOR ALL
  USING (auth.role() = 'service_role');
CREATE POLICY "Users can view own paper_margin_interests"
  ON public.paper_margin_interests FOR SELECT
  USING (auth.uid() = user_id);

CREATE INDEX idx_paper_margin_interests_user_asset
  ON public.paper_margin_interests(user_id, asset);

-- paper_margin_liquidations
CREATE TABLE public.paper_margin_liquidations (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID NOT NULL REFERENCES public.user_profiles(id) ON DELETE CASCADE,
  exchange TEXT NOT NULL DEFAULT '',
  symbol TEXT NOT NULL DEFAULT '',
  side TEXT NOT NULL DEFAULT '',
  order_id BIGINT NOT NULL DEFAULT 0,
  price TEXT NOT NULL DEFAULT '0',
  quantity TEXT NOT NULL DEFAULT '0',
  average_price TEXT NOT NULL DEFAULT '0',
  executed_quantity TEXT NOT NULL DEFAULT '0',
  time_in_force TEXT NOT NULL DEFAULT '',
  is_isolated BOOLEAN NOT NULL DEFAULT false,
  time TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE public.paper_margin_liquidations ENABLE ROW LEVEL SECURITY;
CREATE POLICY "Service role manages paper_margin_liquidations"
  ON public.paper_margin_liquidations FOR ALL
  USING (auth.role() = 'service_role');
CREATE POLICY "Users can view own paper_margin_liquidations"
  ON public.paper_margin_liquidations FOR SELECT
  USING (auth.uid() = user_id);

CREATE UNIQUE INDEX paper_margin_liquidations_user_order_uniq
  ON public.paper_margin_liquidations(user_id, order_id);

-- paper_futures_position_risks
CREATE TABLE public.paper_futures_position_risks (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID NOT NULL REFERENCES public.user_profiles(id) ON DELETE CASCADE,
  exchange TEXT NOT NULL DEFAULT '',
  symbol TEXT NOT NULL DEFAULT '',
  position_side TEXT NOT NULL DEFAULT '',
  leverage TEXT NOT NULL DEFAULT '0',
  liquidation_price TEXT NOT NULL DEFAULT '0',
  entry_price TEXT NOT NULL DEFAULT '0',
  mark_price TEXT NOT NULL DEFAULT '0',
  break_even_price TEXT NOT NULL DEFAULT '0',
  position_amount TEXT NOT NULL DEFAULT '0',
  unrealized_pnl TEXT NOT NULL DEFAULT '0',
  notional TEXT NOT NULL DEFAULT '0',
  initial_margin TEXT NOT NULL DEFAULT '0',
  maint_margin TEXT NOT NULL DEFAULT '0',
  position_initial_margin TEXT NOT NULL DEFAULT '0',
  open_order_initial_margin TEXT NOT NULL DEFAULT '0',
  adl TEXT NOT NULL DEFAULT '0',
  margin_asset TEXT NOT NULL DEFAULT '',
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE public.paper_futures_position_risks ENABLE ROW LEVEL SECURITY;
CREATE POLICY "Service role manages paper_futures_position_risks"
  ON public.paper_futures_position_risks FOR ALL
  USING (auth.role() = 'service_role');
CREATE POLICY "Users can view own paper_futures_position_risks"
  ON public.paper_futures_position_risks FOR SELECT
  USING (auth.uid() = user_id);

CREATE UNIQUE INDEX paper_futures_position_risks_user_symbol_side_uniq
  ON public.paper_futures_position_risks(user_id, symbol, position_side);
CREATE INDEX idx_paper_futures_position_risks_user_exchange
  ON public.paper_futures_position_risks(user_id, exchange);

-- =============================================
-- Realtime publication for new tables
-- =============================================
ALTER PUBLICATION supabase_realtime ADD TABLE public.nav_history_details;
ALTER PUBLICATION supabase_realtime ADD TABLE public.rewards;
ALTER PUBLICATION supabase_realtime ADD TABLE public.withdraws;
ALTER PUBLICATION supabase_realtime ADD TABLE public.deposits;
ALTER PUBLICATION supabase_realtime ADD TABLE public.margin_loans;
ALTER PUBLICATION supabase_realtime ADD TABLE public.margin_repays;
ALTER PUBLICATION supabase_realtime ADD TABLE public.margin_interests;
ALTER PUBLICATION supabase_realtime ADD TABLE public.margin_liquidations;
ALTER PUBLICATION supabase_realtime ADD TABLE public.futures_position_risks;
ALTER PUBLICATION supabase_realtime ADD TABLE public.paper_nav_history_details;
ALTER PUBLICATION supabase_realtime ADD TABLE public.paper_rewards;
ALTER PUBLICATION supabase_realtime ADD TABLE public.paper_withdraws;
ALTER PUBLICATION supabase_realtime ADD TABLE public.paper_deposits;
ALTER PUBLICATION supabase_realtime ADD TABLE public.paper_margin_loans;
ALTER PUBLICATION supabase_realtime ADD TABLE public.paper_margin_repays;
ALTER PUBLICATION supabase_realtime ADD TABLE public.paper_margin_interests;
ALTER PUBLICATION supabase_realtime ADD TABLE public.paper_margin_liquidations;
ALTER PUBLICATION supabase_realtime ADD TABLE public.paper_futures_position_risks;
