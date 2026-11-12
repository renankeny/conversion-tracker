-- Migração 0056: Adiciona trigger_event_ctwa para disparar eventos específicos de WhatsApp/CTWA no Kanban
ALTER TABLE crm_stages
    ADD COLUMN IF NOT EXISTS trigger_event_ctwa TEXT NOT NULL DEFAULT '';

