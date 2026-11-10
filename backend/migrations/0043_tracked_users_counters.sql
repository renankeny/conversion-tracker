-- Migration 0043: Contadores de Visitas, Checkouts e Ultima Atividade em TrackedUsers

ALTER TABLE tracked_users ADD COLUMN IF NOT EXISTS pageview_count INT NOT NULL DEFAULT 1;
ALTER TABLE tracked_users ADD COLUMN IF NOT EXISTS checkout_count INT NOT NULL DEFAULT 0;
ALTER TABLE tracked_users ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
