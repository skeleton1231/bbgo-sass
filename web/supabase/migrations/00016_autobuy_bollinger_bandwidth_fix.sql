-- Fix autobuy: bollinger defaults missing bandWidth, which is required by Validate()
UPDATE strategy_registry SET defaults = jsonb_set(defaults, '{bollinger}', '{"interval": "1h", "window": 20, "bandWidth": 2.0}'::jsonb)
WHERE id = 'autobuy';
