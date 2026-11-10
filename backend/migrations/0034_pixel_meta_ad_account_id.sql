-- =====================================================================
-- Migration 0034 — Pixel Meta Ad Account ID
-- Adiciona suporte a ID da conta de anúncios da Meta na tabela de pixels
-- =====================================================================

ALTER TABLE pixels ADD COLUMN IF NOT EXISTS meta_ad_account_id TEXT DEFAULT '';
