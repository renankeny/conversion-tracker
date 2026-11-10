-- =====================================================================
-- Migration 0039 — Filtro de eventos da página de Eventos, persistido
-- por empresa e por pixel.
--
-- Mapa JSONB: chave é o pixel_name (ou "" pra "Todos os Pixels"), valor é
-- o array de event_name selecionados pra exibir naquele contexto. Cada
-- empresa tem o seu próprio mapa — independente de quem está logado.
-- =====================================================================

ALTER TABLE accounts ADD COLUMN IF NOT EXISTS events_filter JSONB NOT NULL DEFAULT '{}'::jsonb;
