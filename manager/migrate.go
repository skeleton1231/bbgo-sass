package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const migrationSQL = `
CREATE TABLE IF NOT EXISTS public.user_containers (
  user_id UUID PRIMARY KEY REFERENCES public.user_profiles(id) ON DELETE CASCADE,
  status TEXT NOT NULL DEFAULT 'stopped' CHECK (status IN ('running', 'stopped', 'error')),
  strategies JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE public.user_containers ENABLE ROW LEVEL SECURITY;

DO $$ BEGIN
  CREATE POLICY "Service role manages user_containers"
    ON public.user_containers FOR ALL
    USING (auth.role() = 'service_role');
  CREATE POLICY "Users can view own container"
    ON public.user_containers FOR SELECT
    USING (auth.uid() = user_id);
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
  CREATE TRIGGER set_updated_at
    BEFORE UPDATE ON public.user_containers
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

ALTER TABLE public.sync_orders ALTER COLUMN bot_id DROP NOT NULL;
ALTER TABLE public.sync_trades ALTER COLUMN bot_id DROP NOT NULL;

DO $ BEGIN
    ALTER TABLE public.sync_orders ADD COLUMN executed_quantity TEXT;
    ALTER TABLE public.sync_orders ADD COLUMN creation_time TEXT;
    ALTER TABLE public.sync_trades ADD COLUMN quote_quantity TEXT;
    ALTER TABLE public.sync_trades ADD COLUMN traded_at TEXT;
EXCEPTION WHEN duplicate_column THEN NULL;
END $;

CREATE TABLE IF NOT EXISTS public.sync_cursors (
  user_id UUID NOT NULL REFERENCES public.user_profiles(id) ON DELETE CASCADE,
  table_name TEXT NOT NULL,
  last_gid BIGINT NOT NULL DEFAULT 0,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, table_name)
);

ALTER TABLE public.sync_cursors ENABLE ROW LEVEL SECURITY;

DO $$ BEGIN
  CREATE POLICY "Service role manages sync_cursors"
    ON public.sync_cursors FOR ALL
    USING (auth.role() = 'service_role');
  CREATE POLICY "Users can view own cursors"
    ON public.sync_cursors FOR SELECT
    USING (auth.uid() = user_id);
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS public.exchange_credentials (
  id TEXT PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES public.user_profiles(id) ON DELETE CASCADE,
  exchange TEXT NOT NULL,
  api_key_encrypted TEXT NOT NULL,
  api_secret_encrypted TEXT NOT NULL,
  passphrase_encrypted TEXT,
  is_testnet BOOLEAN NOT NULL DEFAULT false,
  is_verified BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(user_id, exchange)
);

ALTER TABLE public.exchange_credentials ENABLE ROW LEVEL SECURITY;

DO $$ BEGIN
  CREATE POLICY "Service role manages exchange_credentials"
    ON public.exchange_credentials FOR ALL
    USING (auth.role() = 'service_role');
  CREATE POLICY "Users can view own credentials"
    ON public.exchange_credentials FOR SELECT
    USING (auth.uid() = user_id);
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
`

func RunMigration(dbURL string) error {
	if dbURL == "" {
		log.Println("SUPABASE_DB_URL not set, skipping auto-migration")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}

	if _, err := db.ExecContext(ctx, migrationSQL); err != nil {
		return fmt.Errorf("run migration: %w", err)
	}

	log.Println("auto-migration completed successfully")
	return nil
}
