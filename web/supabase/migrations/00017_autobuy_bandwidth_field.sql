-- Add missing bollinger.bandWidth field to autobuy strategy
-- bbgo Validate() requires bandWidth > 0, without this field it defaults to 0 and crashes
UPDATE strategy_registry SET fields = fields::jsonb || '[
  {"key": "bollinger.bandWidth", "min": 0.1, "type": "number", "group": "bollinger", "label": "Bollinger Band Width", "default": 2.0}
]'::jsonb
WHERE id = 'autobuy';
