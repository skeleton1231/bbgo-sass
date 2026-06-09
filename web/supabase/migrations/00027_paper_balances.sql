-- Paper trading balance snapshots for full state recovery.
-- Written by bbgo on every balance change, read on container startup.
-- Uses the same RLS pattern as paper_orders/paper_trades.
CREATE TABLE IF NOT EXISTS paper_balances (
    user_id       UUID NOT NULL REFERENCES auth.users(id),
    currency      TEXT NOT NULL,
    available     TEXT NOT NULL DEFAULT '0',
    locked        TEXT NOT NULL DEFAULT '0',
    updated_at    TIMESTAMPTZ DEFAULT now(),
    PRIMARY KEY (user_id, currency)
);

-- RLS
ALTER TABLE paper_balances ENABLE ROW LEVEL SECURITY;

CREATE POLICY "Users can read own paper balances"
    ON paper_balances FOR SELECT
    USING (auth.uid() = user_id);

CREATE POLICY "Service role can do everything"
    ON paper_balances FOR ALL
    USING (true)
    WITH CHECK (true);

-- Realtime publication
ALTER PUBLICATION supabase_realtime ADD TABLE paper_balances;
