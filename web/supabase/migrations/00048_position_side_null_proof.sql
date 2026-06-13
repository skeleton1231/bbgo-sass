-- =============================================
-- Final cleanup for position_side anomalies that 00046 missed
--
-- Background:
--   00046_normalize_one_way_position_side.sql used
--     WHERE position_side IN ('', 'Long', 'Short')
--   but `NULL IN (...)` evaluates to NULL (treated as false in WHERE), so any
--   pre-existing NULL rows were left untouched. Combined with the engine fix
--   in f3dc7828c (always emits BOTH), this migration closes the last gap:
--     1. Normalize NULL -> BOTH (00046 could not reach these)
--     2. Re-run the ''/Long/Short normalization for any rows written between
--        00046 applying and the engine fix landing in running containers
--     3. Add CHECK constraints so the DB rejects future bad writes regardless
--        of what the application does
--
-- Scope:
--   paper_futures_position_risks - engine only supports one-way mode, so the
--     only legal value is 'BOTH'.
--   futures_position_risks       - live mode may legitimately see LONG/SHORT
--     from real exchange hedge-mode APIs, so we only forbid NULL/'' here.
-- =============================================

-- 1. paper_futures_position_risks: nuke NULL + re-normalize stragglers
UPDATE public.paper_futures_position_risks
   SET position_side = 'BOTH'
 WHERE position_side IS NULL
    OR position_side = ''
    OR position_side IN ('Long', 'Short');

ALTER TABLE public.paper_futures_position_risks
  DROP CONSTRAINT IF EXISTS paper_fpr_position_side_chk;
ALTER TABLE public.paper_futures_position_risks
  ADD CONSTRAINT paper_fpr_position_side_chk
    CHECK (position_side = 'BOTH');

-- 2. futures_position_risks (live): only forbid NULL/'' (preserve hedge-mode LONG/SHORT)
UPDATE public.futures_position_risks
   SET position_side = 'BOTH'
 WHERE position_side IS NULL OR position_side = '';

ALTER TABLE public.futures_position_risks
  DROP CONSTRAINT IF EXISTS fpr_position_side_chk;
ALTER TABLE public.futures_position_risks
  ADD CONSTRAINT fpr_position_side_chk
    CHECK (position_side IS NOT NULL AND position_side <> '');
