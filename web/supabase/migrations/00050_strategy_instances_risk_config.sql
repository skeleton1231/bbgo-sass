-- Add risk_config column for per-instance universal risk parameters
-- (stop loss, take profit, ROI thresholds, trailing stop, max position).
-- These are enforced by UniversalRiskController in bbgo at the
-- GeneralOrderExecutor layer, so they apply to ANY strategy.
alter table strategy_instances
  add column if not exists risk_config jsonb null default null;

comment on column strategy_instances.risk_config is 'Universal risk config: {"stopLossPrice":19000,"takeProfitPrice":22000,"roiStopLoss":0.05,"roiTakeProfit":0.10,"trailingActivation":0.05,"trailingCallback":0.02,"maxPositionQty":5}';
