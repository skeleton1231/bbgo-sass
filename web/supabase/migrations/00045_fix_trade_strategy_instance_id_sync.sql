-- Backfill strategy_instance_id on trades and orders after fixing
-- pkg/service/trade.go ON CONFLICT clause (which previously did NOT refresh
-- strategy_instance_id on upsert, leaving stale/empty values on trades while
-- orders were updated).
--
-- Strategy: for each (user_id, order_id), pick the latest non-empty
-- strategy_instance_id among trades (by traded_at). Then:
--   1. Propagate it to every sibling trade under the same order
--   2. Propagate it to the parent order
--
-- Idempotent: WHERE clause skips rows already aligned.
-- Trades table has no updated_at column (only traded_at), so we don't bump it.
--
-- PERFORMANCE: Each of the 4 UPDATE statements below performs a full table scan
-- plus a sort over trades/paper_trades (DISTINCT ON ... ORDER BY traded_at DESC).
-- Run during a maintenance window on tenants with millions of trade rows, or
-- batch by user_id before pushing this migration.
--
-- TIE-BREAK: When multiple trades under the same (user_id, order_id) have
-- strategy_instance_id set but identical (or all-NULL) traded_at, the row chosen
-- by DISTINCT ON is implementation-defined. In practice this should be rare
-- (traded_at is normally populated by the exchange feed); flag any rows that
-- still look wrong after the backfill and patch them manually.

-- =============================================
-- LIVE trades: align sibling trades to latest non-empty SID per order
-- =============================================
UPDATE trades t
SET strategy_instance_id = latest.sid
FROM (
  SELECT DISTINCT ON (user_id, order_id)
         user_id, order_id, strategy_instance_id AS sid
  FROM trades
  WHERE COALESCE(strategy_instance_id, '') <> ''
  ORDER BY user_id, order_id, traded_at DESC NULLS LAST
) latest
WHERE t.user_id = latest.user_id
  AND t.order_id = latest.order_id
  AND COALESCE(t.strategy_instance_id, '') <> COALESCE(latest.sid, '');

-- =============================================
-- LIVE orders: align order SID to latest trade SID
-- =============================================
UPDATE orders o
SET strategy_instance_id = latest.sid,
    updated_at = NOW()
FROM (
  SELECT DISTINCT ON (user_id, order_id)
         user_id, order_id, strategy_instance_id AS sid
  FROM trades
  WHERE COALESCE(strategy_instance_id, '') <> ''
  ORDER BY user_id, order_id, traded_at DESC NULLS LAST
) latest
WHERE o.user_id = latest.user_id
  AND o.order_id = latest.order_id
  AND COALESCE(o.strategy_instance_id, '') <> COALESCE(latest.sid, '');

-- =============================================
-- PAPER trades: same as live
-- =============================================
UPDATE paper_trades t
SET strategy_instance_id = latest.sid
FROM (
  SELECT DISTINCT ON (user_id, order_id)
         user_id, order_id, strategy_instance_id AS sid
  FROM paper_trades
  WHERE COALESCE(strategy_instance_id, '') <> ''
  ORDER BY user_id, order_id, traded_at DESC NULLS LAST
) latest
WHERE t.user_id = latest.user_id
  AND t.order_id = latest.order_id
  AND COALESCE(t.strategy_instance_id, '') <> COALESCE(latest.sid, '');

-- =============================================
-- PAPER orders: same as live
-- =============================================
UPDATE paper_orders o
SET strategy_instance_id = latest.sid,
    updated_at = NOW()
FROM (
  SELECT DISTINCT ON (user_id, order_id)
         user_id, order_id, strategy_instance_id AS sid
  FROM paper_trades
  WHERE COALESCE(strategy_instance_id, '') <> ''
  ORDER BY user_id, order_id, traded_at DESC NULLS LAST
) latest
WHERE o.user_id = latest.user_id
  AND o.order_id = latest.order_id
  AND COALESCE(o.strategy_instance_id, '') <> COALESCE(latest.sid, '');
