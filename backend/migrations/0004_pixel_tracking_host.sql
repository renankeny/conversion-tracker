-- 0004_pixel_tracking_host.sql
--
-- Adiciona coluna tracking_host à tabela pixels.
--
-- Quando preenchido, o snippet gerado para o cliente aponta para esse
-- subdomínio em vez do host padrão da API. O cliente precisa configurar
-- um CNAME no DNS dele apontando esse subdomínio para o servidor TrackFind.
-- Isso garante cookies first-party e contorna boa parte dos adblockers.
--
-- Exemplo: cliente compra "tracking.cliente.com" e configura:
--   tracking.cliente.com.  CNAME  track.trackfind.com.br.
--
-- Idempotente: usa IF NOT EXISTS.

ALTER TABLE pixels
    ADD COLUMN IF NOT EXISTS tracking_host TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_pixels_tracking_host ON pixels(tracking_host)
    WHERE tracking_host <> '';

COMMENT ON COLUMN pixels.tracking_host IS
    'Subdomínio do cliente (ex: tracking.cliente.com) apontando via CNAME para o servidor TrackFind. Quando vazio, usa o host padrão da API.';
