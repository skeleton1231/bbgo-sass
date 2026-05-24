-- Add columns that the Go sync code inserts but were missing from the initial schema

alter table public.sync_orders
  add column if not exists executed_quantity text,
  add column if not exists creation_time text;

alter table public.sync_trades
  add column if not exists quote_quantity text,
  add column if not exists traded_at text;
