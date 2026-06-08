-- Enable Supabase Realtime for trading data tables
-- Realtime respects RLS: users only receive events for their own rows (auth.uid() = user_id)

-- Live trading tables
ALTER PUBLICATION supabase_realtime ADD TABLE public.orders;
ALTER PUBLICATION supabase_realtime ADD TABLE public.trades;
ALTER PUBLICATION supabase_realtime ADD TABLE public.positions;
ALTER PUBLICATION supabase_realtime ADD TABLE public.profits;

-- Paper trading tables
ALTER PUBLICATION supabase_realtime ADD TABLE public.paper_orders;
ALTER PUBLICATION supabase_realtime ADD TABLE public.paper_trades;
ALTER PUBLICATION supabase_realtime ADD TABLE public.paper_positions;
ALTER PUBLICATION supabase_realtime ADD TABLE public.paper_profits;
