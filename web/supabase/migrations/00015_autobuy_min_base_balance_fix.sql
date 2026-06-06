-- Fix autobuy minBaseBalance default: bbgo Validate() requires > 0
UPDATE strategy_registry SET defaults = jsonb_set(defaults, '{minBaseBalance}', '0.001')
WHERE id = 'autobuy';
