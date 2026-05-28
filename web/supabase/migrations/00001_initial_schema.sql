-- Enable UUID extension
create extension if not exists "uuid-ossp";

-- =============================================
-- User profiles (extends Supabase auth.users)
-- =============================================
create table public.user_profiles (
  id uuid primary key references auth.users(id) on delete cascade,
  email text not null,
  display_name text,
  role text not null default 'user',
  avatar_url text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

alter table public.user_profiles enable row level security;

create policy "Users can view own profile"
  on public.user_profiles for select
  using (auth.uid() = id);

create policy "Users can update own profile"
  on public.user_profiles for update
  using (auth.uid() = id);

-- Auto-create profile on signup
create or replace function public.handle_new_user()
returns trigger as $$
begin
  insert into public.user_profiles (id, email, display_name)
  values (new.id, new.email, coalesce(new.raw_user_meta_data->>'display_name', split_part(new.email, '@', 1)));
  return new;
end;
$$ language plpgsql security definer;

create or replace trigger on_auth_user_created
  after insert on auth.users
  for each row execute function public.handle_new_user();

-- =============================================
-- Exchange API credentials (encrypted)
-- =============================================
create table public.exchange_credentials (
  id uuid primary key default uuid_generate_v4(),
  user_id uuid not null references public.user_profiles(id) on delete cascade,
  exchange text not null,
  api_key_encrypted text not null,
  api_secret_encrypted text not null,
  passphrase_encrypted text,
  is_testnet boolean not null default false,
  is_verified boolean not null default false,
  last_verified_at timestamptz,
  created_at timestamptz not null default now()
);

alter table public.exchange_credentials enable row level security;

create policy "Users can manage own credentials"
  on public.exchange_credentials for all
  using (auth.uid() = user_id);

create index idx_exchange_credentials_user on public.exchange_credentials(user_id);

-- =============================================
-- Trading bots
-- =============================================
create table public.bots (
  id uuid primary key default uuid_generate_v4(),
  user_id uuid not null references public.user_profiles(id) on delete cascade,
  name text not null,
  exchange text not null,
  strategy text not null,
  config jsonb not null default '{}',
  mode text not null default 'paper' check (mode in ('live', 'paper')),
  status text not null default 'stopped' check (status in ('running', 'stopped', 'error')),
  bbgo_pid integer,
  webserver_port integer,
  grpc_port integer,
  config_path text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

alter table public.bots enable row level security;

create policy "Users can manage own bots"
  on public.bots for all
  using (auth.uid() = user_id);

create index idx_bots_user on public.bots(user_id);
create index idx_bots_status on public.bots(status);

-- =============================================
-- Synced orders (from BBGO SQLite)
-- =============================================
create table public.sync_orders (
  id uuid primary key default uuid_generate_v4(),
  bot_id uuid not null references public.bots(id) on delete cascade,
  user_id uuid not null references public.user_profiles(id) on delete cascade,
  symbol text not null,
  side text not null,
  type text not null,
  price text not null,
  quantity text not null,
  status text not null,
  order_id text not null,
  synced_at timestamptz not null default now(),
  created_at timestamptz not null default now()
);

alter table public.sync_orders enable row level security;

create policy "Users can view own orders"
  on public.sync_orders for select
  using (auth.uid() = user_id);

create index idx_sync_orders_bot on public.sync_orders(bot_id);
create index idx_sync_orders_user on public.sync_orders(user_id);

-- =============================================
-- Synced trades (from BBGO SQLite)
-- =============================================
create table public.sync_trades (
  id uuid primary key default uuid_generate_v4(),
  bot_id uuid not null references public.bots(id) on delete cascade,
  user_id uuid not null references public.user_profiles(id) on delete cascade,
  symbol text not null,
  side text not null,
  price text not null,
  quantity text not null,
  fee text not null,
  fee_currency text not null,
  trade_id text not null,
  order_id text not null,
  pnl text,
  synced_at timestamptz not null default now(),
  created_at timestamptz not null default now()
);

alter table public.sync_trades enable row level security;

create policy "Users can view own trades"
  on public.sync_trades for select
  using (auth.uid() = user_id);

create index idx_sync_trades_bot on public.sync_trades(bot_id);
create index idx_sync_trades_user on public.sync_trades(user_id);

-- =============================================
-- Sync cursors (track sync progress per user)
-- =============================================
create table public.sync_cursors (
  user_id uuid not null references public.user_profiles(id) on delete cascade,
  table_name text not null,
  last_gid bigint not null default 0,
  updated_at timestamptz not null default now(),
  primary key (user_id, table_name)
);

alter table public.sync_cursors enable row level security;

create policy "Service role manages cursors"
  on public.sync_cursors for all
  using (auth.role() = 'service_role');

-- =============================================
-- Backtest reports
-- =============================================
create table public.backtest_reports (
  id uuid primary key default uuid_generate_v4(),
  user_id uuid not null references public.user_profiles(id) on delete cascade,
  strategy text not null,
  config jsonb not null default '{}',
  start_date date not null,
  end_date date not null,
  total_profit text not null,
  max_drawdown text not null,
  sharpe_ratio text,
  sortino_ratio text,
  profit_factor text,
  win_rate text not null,
  total_trades integer not null,
  win_count integer not null,
  loss_count integer not null,
  cagr text,
  report_json jsonb not null default '{}',
  created_at timestamptz not null default now()
);

alter table public.backtest_reports enable row level security;

create policy "Users can manage own backtest reports"
  on public.backtest_reports for all
  using (auth.uid() = user_id);

create index idx_backtest_reports_user on public.backtest_reports(user_id);

-- =============================================
-- Updated_at trigger function
-- =============================================
create or replace function public.update_updated_at()
returns trigger as $$
begin
  new.updated_at = now();
  return new;
end;
$$ language plpgsql;

create trigger set_updated_at
  before update on public.user_profiles
  for each row execute function public.update_updated_at();

create trigger set_updated_at
  before update on public.bots
  for each row execute function public.update_updated_at();
