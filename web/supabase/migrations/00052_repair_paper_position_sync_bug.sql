-- Repair: paper-futures strategy position sync bug.
--
-- Before the code fix in pkg/bbgo/order_executor_general.go
-- (SyncStrategyPositionFromFuturesState), a SPOT strategy running on a
-- FUTURES paper session started every container restart with
-- s.Position.Base=0 / AverageCost=0 (stale or missing JSON persistence),
-- even though paperFuturesState was correctly restored from
-- paper_futures_position_risks. Spot AddTrade then misclassified
-- reducing trades as opens, producing wrong snapshots:
--   * paper_profits.profit / net_profit computed from wrong Base/AverageCost
--     (typical symptom: 0.10-0.40 USDT per close instead of correct values)
--   * paper_positions.base / average_cost accumulating wrong state
--
-- Two tables remain authoritative and were NEVER affected:
--   * paper_trades.pnl — set by computeRealizedPnLLocked in the matching
--     engine, which reads paperFuturesState directly.
--   * paper_futures_position_risks — written by FuturesService from
--     QueryPositionRisk(), also paperFuturesState-driven.
--
-- This migration is idempotent: re-running it is a no-op on healthy
-- rows (values already match) and on rows with no futures pnl
-- (filtered out). It runs across all users/instances — any paper bot
-- that hit this bug gets repaired, not just the one that surfaced it.

-- 1. Recompute paper_profits.profit / net_profit from paper_trades.pnl.
--
--    paper_trades.pnl is non-NULL/non-zero only for futures reducing
--    trades (set inside the `isFutures` branch of buildFillLocked),
--    so this UPDATE is implicitly scoped to futures and skips spot.
--    Each paper_profits row maps 1:1 to a reducing trade via trade_id.
UPDATE public.paper_profits p
SET    profit = t.pnl,
       net_profit = (t.pnl_num - t.fee_num)::text,
       profit_margin = CASE
         WHEN t.price_num = 0 OR t.qty_num = 0 THEN '0'
         ELSE (t.pnl_num / (t.price_num * t.qty_num))::text
       END,
       net_profit_margin = CASE
         WHEN t.price_num = 0 OR t.qty_num = 0 THEN '0'
         ELSE ((t.pnl_num - t.fee_num) / (t.price_num * t.qty_num))::text
       END
FROM   (
  SELECT user_id,
         strategy_instance_id,
         trade_id,
         -- Normalise text → numeric once per row.
         pnl::numeric                                                      AS pnl_num,
         TRIM(COALESCE(pnl, '0'))::text                                    AS pnl,
         TRIM(COALESCE(fee, '0'))::numeric                                 AS fee_num,
         TRIM(COALESCE(price, '0'))::numeric                               AS price_num,
         TRIM(COALESCE(quantity, '0'))::numeric                            AS qty_num
  FROM   public.paper_trades
  WHERE  pnl IS NOT NULL
    AND  pnl != '0'
) t
WHERE  p.trade_id::text = t.trade_id
  AND  p.user_id = t.user_id
  AND  p.strategy_instance_id = t.strategy_instance_id
  -- Idempotent guard: only touch rows where the stored profit
  -- diverges from the authoritative pnl. Healthy bots skip entirely.
  AND  TRIM(COALESCE(p.profit, '0'))::numeric IS DISTINCT FROM t.pnl_num;

-- 2. Patch the LATEST paper_positions snapshot per
--    (user_id, strategy_instance_id, symbol) to match the latest
--    paper_futures_position_risks.
--
--    Historical snapshots are intentionally NOT rewritten — they
--    reflect (wrong) s.Position state at the time, and replaying
--    every trade against corrected state is brittle. The dashboard
--    reads "current position" from paper_futures_position_risks via
--    useUnrealizedPnL, so paper_positions is secondary; patching the
--    newest row is enough for any consumer that reads the latest
--    snapshot. After the code fix ships, new snapshots land correctly.
UPDATE public.paper_positions pp
SET    base = fpr.position_amount,
       average_cost = fpr.entry_price
FROM   (
  SELECT DISTINCT ON (user_id, strategy_instance_id, symbol)
         user_id,
         strategy_instance_id,
         symbol,
         position_amount,
         entry_price
  FROM   public.paper_futures_position_risks
  WHERE  position_amount IS NOT NULL
    AND  position_amount != '0'
  ORDER  BY user_id, strategy_instance_id, symbol, updated_at DESC
) fpr
WHERE  pp.user_id = fpr.user_id
  AND  pp.strategy_instance_id = fpr.strategy_instance_id
  AND  pp.symbol = fpr.symbol
  AND  pp.traded_at = (
    SELECT MAX(p2.traded_at)
    FROM   public.paper_positions p2
    WHERE  p2.user_id = pp.user_id
      AND  p2.strategy_instance_id = pp.strategy_instance_id
      AND  p2.symbol = pp.symbol
  )
  -- Idempotent guard: only patch when current values diverge.
  AND  (
    TRIM(COALESCE(pp.base, '0'))::numeric IS DISTINCT FROM
      TRIM(COALESCE(fpr.position_amount, '0'))::numeric
    OR TRIM(COALESCE(pp.average_cost, '0'))::numeric IS DISTINCT FROM
      TRIM(COALESCE(fpr.entry_price, '0'))::numeric
  );

-- NOTE: After applying this migration, the bbgo container image must be
-- rebuilt and existing paper-futures containers recreated so the code
-- fix in NewGeneralOrderExecutor takes effect — otherwise new fills
-- will continue to write wrong paper_profits/paper_positions and this
-- migration will need to be re-run (it is safe to do so).
