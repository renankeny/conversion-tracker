-- 0026_crm_kanban.sql

CREATE TABLE IF NOT EXISTS crm_stages (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    color TEXT NOT NULL DEFAULT '#E5E7EB',
    order_index INT NOT NULL DEFAULT 0,
    trigger_event TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_crm_stages_account ON crm_stages(account_id);

CREATE TABLE IF NOT EXISTS crm_leads (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    tracked_user_id BIGINT NOT NULL REFERENCES tracked_users(id) ON DELETE CASCADE,
    stage_id BIGINT REFERENCES crm_stages(id) ON DELETE SET NULL,
    value NUMERIC(10,2) NOT NULL DEFAULT 0.00,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(account_id, tracked_user_id)
);

CREATE INDEX IF NOT EXISTS idx_crm_leads_account ON crm_leads(account_id);
CREATE INDEX IF NOT EXISTS idx_crm_leads_stage ON crm_leads(stage_id);

CREATE TABLE IF NOT EXISTS crm_tags (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    color TEXT NOT NULL DEFAULT '#E5E7EB',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_crm_tags_account ON crm_tags(account_id);

CREATE TABLE IF NOT EXISTS crm_lead_tags (
    crm_lead_id BIGINT NOT NULL REFERENCES crm_leads(id) ON DELETE CASCADE,
    crm_tag_id BIGINT NOT NULL REFERENCES crm_tags(id) ON DELETE CASCADE,
    PRIMARY KEY (crm_lead_id, crm_tag_id)
);

CREATE TABLE IF NOT EXISTS crm_notes (
    id BIGSERIAL PRIMARY KEY,
    crm_lead_id BIGINT NOT NULL REFERENCES crm_leads(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_crm_notes_lead ON crm_notes(crm_lead_id);
