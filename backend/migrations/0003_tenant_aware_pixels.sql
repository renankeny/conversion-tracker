-- Multi-tenant: cada pixel passa a ter uma lista de domínios autorizados
-- (CORS / Origin) e uma configuração JSON do tracker que é injetada inline
-- no /tracker.js servido pelo backend.

ALTER TABLE pixels
    ADD COLUMN IF NOT EXISTS domains TEXT[] NOT NULL DEFAULT '{}';

ALTER TABLE pixels
    ADD COLUMN IF NOT EXISTS tracker_config JSONB NOT NULL DEFAULT '{}'::jsonb;

-- Index pra resolver pixel a partir do hostname da request quando precisar
-- (ex: validar Origin). Usa GIN pra buscas com `domains @> ARRAY[...]`.
CREATE INDEX IF NOT EXISTS idx_pixels_domains ON pixels USING GIN(domains);
