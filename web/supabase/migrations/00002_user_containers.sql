-- User-centric container tracking (replaces per-bot tracking)
create table public.user_containers (
  user_id uuid primary key references public.user_profiles(id) on delete cascade,
  status text not null default 'stopped' check (status in ('running', 'stopped', 'error')),
  strategies jsonb not null default '[]'::jsonb,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

alter table public.user_containers enable row level security;

create policy "Service role manages user_containers"
  on public.user_containers for all
  using (auth.role() = 'service_role');

create policy "Users can view own container"
  on public.user_containers for select
  using (auth.uid() = user_id);

create trigger set_updated_at
  before update on public.user_containers
  for each row execute function public.update_updated_at();

-- Make bot_id nullable in sync tables (user-centric model doesn't always have a bot_id)
alter table public.sync_orders alter column bot_id drop not null;
alter table public.sync_trades alter column bot_id drop not null;
