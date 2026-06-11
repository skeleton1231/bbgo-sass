-- Remove requires_futures from margin-only strategies (margin not supported yet)
UPDATE strategy_registry SET requires_futures = false WHERE id = 'autoborrow';
UPDATE strategy_registry SET requires_futures = false WHERE id = 'deposit2transfer';
