-- =============================================
-- Normalize position_side to 'BOTH' for one-way mode positions
--
-- Background:
--   The bbgo paper trade engine (pkg/bbgo/paper_trade_futures.go) only supports
--   one-way mode but was previously labeling snapshots with "Long"/"Short" based
--   on position_amount sign, and resetting to "" on close. This caused closed/flip
--   snapshots to land on a different (exchange, symbol, position_side) bucket than
--   the open snapshot, leaving stale open rows in the DB forever (C2 root cause).
--
-- After the engine fix, new paper snapshots always write position_side='BOTH'.
-- This migration normalizes existing rows so historical data aligns with the
-- new convention and the frontend can stop treating "" as a special bucket.
--
-- Scope:
--   paper_futures_position_risks: normalize ''/'Long'/'Short' → 'BOTH'
--     (paper engine has never supported hedge mode — all rows are one-way)
--   futures_position_risks:       normalize only '' → 'BOTH'
--     (live mode may legitimately have hedge-mode Long/Short rows from real
--      exchange APIs — preserve those)
-- =============================================

UPDATE public.paper_futures_position_risks
   SET position_side = 'BOTH'
 WHERE position_side IN ('', 'Long', 'Short');

UPDATE public.futures_position_risks
   SET position_side = 'BOTH'
 WHERE position_side = '';
