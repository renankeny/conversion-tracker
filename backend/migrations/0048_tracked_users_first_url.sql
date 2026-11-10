-- =====================================================================
-- 0048_tracked_users_first_url.sql
--
-- Adiciona a coluna first_url em tracked_users para registrar a URL
-- de entrada (descoberta/landing page) da primeira visita do usuário.
-- =====================================================================

ALTER TABLE tracked_users ADD COLUMN IF NOT EXISTS first_url TEXT NOT NULL DEFAULT '';

-- Backfill: inicializa first_url com a url existente nos registros onde first_url estiver vazia
UPDATE tracked_users 
SET first_url = url 
WHERE first_url = '' AND url <> '';
