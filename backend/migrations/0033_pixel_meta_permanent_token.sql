-- =====================================================================
-- Migration 0033 — Pixel Meta Permanent Access Token & WhatsApp Chats Metadata
-- Adiciona suporte a token de acesso permanente da Meta na tabela de pixels
-- Adiciona coluna metadata na tabela whatsapp_chats para persistência de CTWA
-- =====================================================================

ALTER TABLE pixels ADD COLUMN IF NOT EXISTS meta_permanent_token TEXT DEFAULT '';

ALTER TABLE whatsapp_chats ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}';
