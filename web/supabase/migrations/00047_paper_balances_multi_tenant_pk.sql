-- =============================================
-- Fix paper_balances multi-tenant isolation
-- =============================================
-- Previously PK was (user_id, currency). When two paper bots share a user_id,
-- each bot's upsert overwrote the other's balance row — corrupting both
-- the live state and the RestoreFromDB snapshot on container restart.
--
-- Migration 00034 added strategy_instance_id column but code never used it.
-- This migration promotes strategy_instance_id into the PK so each bot
-- gets its own row. Existing rows (strategy_instance_id='') are preserved
-- under the empty namespace.

-- 1. Drop the old PK
ALTER TABLE public.paper_balances
    DROP CONSTRAINT IF EXISTS paper_balances_pkey;

-- 2. Recreate PK with strategy_instance_id
ALTER TABLE public.paper_balances
    ADD CONSTRAINT paper_balances_pkey
    PRIMARY KEY (user_id, strategy_instance_id, currency);

-- 3. Index for bot-detail queries filtered by strategy_instance_id
CREATE INDEX IF NOT EXISTS idx_paper_balances_strategy_instance
    ON public.paper_balances(user_id, strategy_instance_id);
