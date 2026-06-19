-- Track per-instance container startup failures so the manager can surface
-- crashloop / strategy-validation errors to users instead of reporting a
-- phantom-active container. Cleared on every successful start.
alter table strategy_instances
  add column if not exists last_error text null default null,
  add column if not exists last_error_at timestamptz null default null;

comment on column strategy_instances.last_error is 'Last captured container error (level=fatal / cannot execute command) extracted from docker logs when the container failed to start or entered a crashloop. NULL when the instance is healthy.';
comment on column strategy_instances.last_error_at is 'UTC timestamp when last_error was recorded.';
