-- Pivotshort: add default exit method (roiStopLoss) to prevent unlimited losses
UPDATE strategy_registry SET
  defaults = '{
    "interval": "1h", "quantity": 0.001, "leverage": 1,
    "breakLow": {"interval": "1h", "window": 7, "ratio": 0.01},
    "exits": [{"roiStopLoss": {"percentage": -0.05}}]
  }'::jsonb
WHERE id = 'pivotshort';
