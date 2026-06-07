-- Add requires_futures column: strategies that need futures/margin session
-- because they sell (short) without holding base currency.
alter table strategy_registry
  add column requires_futures boolean null default false;

-- Pure short strategies: OpenPosition with Short=true
update strategy_registry set requires_futures = true
where id in ('pivotshort', 'drift', 'elliottwave');

-- Bidirectional market makers: submit SELL LimitMaker without holding base
update strategy_registry set requires_futures = true
where id in ('bollmaker', 'linregmaker', 'rsmaker', 'fixedmaker', 'audacitymaker');
