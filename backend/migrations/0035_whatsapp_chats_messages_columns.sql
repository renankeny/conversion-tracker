-- =====================================================================
-- Migration 0035 — Colunas faltantes em whatsapp_chats/whatsapp_messages
-- O código do repositório (whatsapp_repo.go) já lê/escreve contact_name,
-- profile_pic, is_from_me, has_media, media_url, media_mime_type — essas
-- colunas nunca chegaram a ser criadas na migração 0031. Sem elas, todo
-- INSERT/SELECT feito pelo webhook da Evolution GO falha e a mensagem é
-- descartada (nenhuma conversa fica persistida).
-- =====================================================================

ALTER TABLE whatsapp_chats ADD COLUMN IF NOT EXISTS contact_name TEXT;
ALTER TABLE whatsapp_chats ADD COLUMN IF NOT EXISTS profile_pic TEXT;

UPDATE whatsapp_chats SET contact_name = NULLIF(push_name, '') WHERE contact_name IS NULL;

ALTER TABLE whatsapp_messages ADD COLUMN IF NOT EXISTS is_from_me BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE whatsapp_messages ADD COLUMN IF NOT EXISTS has_media BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE whatsapp_messages ADD COLUMN IF NOT EXISTS media_url TEXT;
ALTER TABLE whatsapp_messages ADD COLUMN IF NOT EXISTS media_mime_type TEXT;

UPDATE whatsapp_messages SET is_from_me = from_me;
