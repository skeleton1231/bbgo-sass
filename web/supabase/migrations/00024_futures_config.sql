-- Add futures_config column for per-instance futures/margin settings
alter table strategy_instances
  add column if not exists futures_config jsonb null default null;

comment on column strategy_instances.futures_config is 'Futures/margin config: {"leverage":3,"marginType":"cross"}';
