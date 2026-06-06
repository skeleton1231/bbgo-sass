-- Allow anonymous (client-side) reads of strategy_registry
-- The frontend StrategyRegistryProvider queries this table using the anon key
-- to populate the strategy dropdown and config form fields.

CREATE POLICY "Anonymous users can read strategy_registry"
  ON strategy_registry FOR SELECT TO anon USING (true);
