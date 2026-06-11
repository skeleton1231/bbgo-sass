-- Add position_action columns to orders, trades, positions (live + paper)
-- Values: OPEN, ADD, REDUCE, CLOSE (spot)
--         OPEN_LONG, ADD_LONG, REDUCE_LONG, CLOSE_LONG,
--         OPEN_SHORT, ADD_SHORT, REDUCE_SHORT, CLOSE_SHORT,
--         FLIP_LONG_TO_SHORT, FLIP_SHORT_TO_LONG (futures)

-- Live tables
ALTER TABLE public.orders ADD COLUMN IF NOT EXISTS position_action TEXT DEFAULT '';
ALTER TABLE public.trades ADD COLUMN IF NOT EXISTS position_action TEXT DEFAULT '';
ALTER TABLE public.positions ADD COLUMN IF NOT EXISTS position_action TEXT DEFAULT '';
ALTER TABLE public.positions ADD COLUMN IF NOT EXISTS last_position_action TEXT DEFAULT '';

-- Paper tables
ALTER TABLE public.paper_orders ADD COLUMN IF NOT EXISTS position_action TEXT DEFAULT '';
ALTER TABLE public.paper_trades ADD COLUMN IF NOT EXISTS position_action TEXT DEFAULT '';
ALTER TABLE public.paper_positions ADD COLUMN IF NOT EXISTS position_action TEXT DEFAULT '';
ALTER TABLE public.paper_positions ADD COLUMN IF NOT EXISTS last_position_action TEXT DEFAULT '';
