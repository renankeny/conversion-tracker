-- =====================================================================
-- Migration 0055 — Moeda por Conexão WhatsApp (ISO-4217 Oficial Meta CAPI)
-- Permite escolher a moeda da operação (BRL, USD, EUR, GBP, etc.) em cada
-- conexão de WhatsApp para envio nos eventos de conversão.
-- =====================================================================

ALTER TABLE whatsapp_instances
    ADD COLUMN IF NOT EXISTS currency TEXT NOT NULL DEFAULT 'BRL';

