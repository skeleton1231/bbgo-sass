-- Strategy instances: per-instance container tracking
-- Replaces the old file-based instance store for cloud persistence

CREATE TABLE strategy_instances (
    instance_id  text NOT NULL,
    user_id      uuid NOT NULL REFERENCES auth.users(id),
    mode         text NOT NULL CHECK (mode IN ('live', 'paper')),
    strategy     text NOT NULL,
    exchange     text NOT NULL DEFAULT '',
    symbol       text NOT NULL DEFAULT '',
    config       jsonb NOT NULL DEFAULT '{}',
    name         text NOT NULL DEFAULT '',
    cross_exchange boolean NOT NULL DEFAULT false,
    sessions     jsonb DEFAULT '[]',
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, mode, instance_id)
);

ALTER TABLE strategy_instances ENABLE ROW LEVEL SECURITY;

CREATE POLICY "Users see own instances"
    ON strategy_instances FOR SELECT
    USING (auth.uid() = user_id);

CREATE POLICY "Service role full access"
    ON strategy_instances FOR ALL
    TO service_role
    USING (true) WITH CHECK (true);

CREATE INDEX idx_strategy_instances_user_mode
    ON strategy_instances(user_id, mode);
