-- Migration 0044: Classificação de Tráfego (Humano, Bot, Inconclusivo) e Heurísticas

ALTER TABLE tracked_users ADD COLUMN IF NOT EXISTS referrer TEXT NOT NULL DEFAULT '';
ALTER TABLE tracked_users ADD COLUMN IF NOT EXISTS traffic_type VARCHAR(20) NOT NULL DEFAULT 'human';
ALTER TABLE tracked_users ADD COLUMN IF NOT EXISTS traffic_score INT NOT NULL DEFAULT 60;
ALTER TABLE tracked_users ADD COLUMN IF NOT EXISTS traffic_category VARCHAR(30) NOT NULL DEFAULT 'none';
ALTER TABLE tracked_users ADD COLUMN IF NOT EXISTS traffic_rules JSONB NOT NULL DEFAULT '[]'::jsonb;

CREATE INDEX IF NOT EXISTS idx_tracked_users_account_traffic ON tracked_users(account_id, traffic_type);
