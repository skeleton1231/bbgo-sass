-- Complete requires_futures audit: add missing flags for strategies that can go net short
-- or require a futures/margin session.

-- === Single-exchange strategies ===

-- supertrend: uses SideEffectTypeMarginBuy for all orders, can go short on sell signal
UPDATE strategy_registry SET requires_futures = true WHERE id = 'supertrend';

-- harmonic: submits market sell orders ("sharkShort") without checking base balance
UPDATE strategy_registry SET requires_futures = true WHERE id = 'harmonic';

-- autoborrow: requires session.Margin for margin borrow/repay operations
UPDATE strategy_registry SET requires_futures = true WHERE id = 'autoborrow';

-- deposit2transfer: requires session.Margin for margin repay operations (liveOnly)
UPDATE strategy_registry SET requires_futures = true WHERE id = 'deposit2transfer';

-- === Cross-exchange strategies with futures session roles ===

-- xpremium: opens short positions with Short:true, checks session.Futures for leverage sizing
UPDATE strategy_registry SET requires_futures = true WHERE id = 'xpremium';

-- xfunding: funding rate arbitrage between spot and futures, requires futures session
UPDATE strategy_registry SET requires_futures = true WHERE id = 'xfunding';

-- xfundingv2: advanced funding rate arbitrage, opens short futures positions
UPDATE strategy_registry SET requires_futures = true WHERE id = 'xfundingv2';

-- Remove leverage field from newly flagged single-exchange strategies
-- (FuturesConfigFields component handles leverage, avoid duplicate)
UPDATE strategy_registry
SET fields = (
  SELECT jsonb_agg(elem)
  FROM jsonb_array_elements(fields) elem
  WHERE elem->>'key' != 'leverage'
)
WHERE id IN ('supertrend', 'harmonic')
  AND requires_futures = true;
