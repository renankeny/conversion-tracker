-- =====================================================================
-- 0021_pixels_public_token.sql
-- Adiciona public_token aos pixels: identificador opaco de 32 chars hex
-- usado no snippet HTML em vez do pixel_name.
--
-- Benefícios:
--   - Não expõe o nome interno do pixel no HTML do site
--   - Permite revogar e regenerar sem mudar o pixel_name
--
-- Compatibilidade retroativa: loader.js e /events/send continuam
-- aceitando tanto o pixel_name quanto o public_token.
-- =====================================================================

ALTER TABLE pixels
    ADD COLUMN IF NOT EXISTS public_token TEXT NOT NULL DEFAULT '';

-- Backfill: gera token único para pixels já existentes usando md5 de
-- (id + clock_timestamp + random) — adequado para retroalimentação.
-- Novos pixels recebem token gerado em Go (crypto/rand, mais seguro).
UPDATE pixels
SET public_token = md5(id::text || clock_timestamp()::text || random()::text)
WHERE public_token = '';

-- Índice único parcial (exclui strings vazias por defensividade)
CREATE UNIQUE INDEX IF NOT EXISTS uq_pixels_public_token
    ON pixels(public_token)
    WHERE public_token <> '';
