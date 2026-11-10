-- 0030_crm_gads_overrides.sql

ALTER TABLE crm_stages
ADD COLUMN IF NOT EXISTS google_ads_account_id BIGINT REFERENCES google_ads_accounts(id) ON DELETE SET NULL,
ADD COLUMN IF NOT EXISTS google_ads_action_id TEXT NOT NULL DEFAULT '';
