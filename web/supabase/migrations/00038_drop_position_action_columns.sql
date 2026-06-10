-- Drop position_action columns from all tables (feature removed, computed client-side instead)
ALTER TABLE public.orders DROP COLUMN IF EXISTS position_action;
ALTER TABLE public.trades DROP COLUMN IF EXISTS position_action;
ALTER TABLE public.positions DROP COLUMN IF EXISTS position_action;
ALTER TABLE public.paper_orders DROP COLUMN IF EXISTS position_action;
ALTER TABLE public.paper_trades DROP COLUMN IF EXISTS position_action;
ALTER TABLE public.paper_positions DROP COLUMN IF EXISTS position_action;
