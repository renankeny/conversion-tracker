-- 0023_partial_indexes.sql

-- Índice parcial focado APENAS em eventos que falharam.
-- Ideal para queries que listam/filtram a fila de erros, reduzindo a busca
-- de milhões de linhas para apenas as poucas linhas que falharam.
CREATE INDEX IF NOT EXISTS idx_sent_events_failed
    ON sent_events(created_at DESC)
    WHERE status = 'failed';
