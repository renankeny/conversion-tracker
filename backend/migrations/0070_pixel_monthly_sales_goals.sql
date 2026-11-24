-- =====================================================================
-- 0070_pixel_monthly_sales_goals.sql
-- Metas mensais por período (YYYY-MM) em pixels
-- Garante que as colunas base existam antes do backfill (compatível com
-- bancos novos ou anteriores à 0071_pixel_sales_goal.sql).
-- =====================================================================

ALTER TABLE pixels
    ADD COLUMN IF NOT EXISTS sales_goal_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS sales_goal_type VARCHAR(16) NOT NULL DEFAULT 'revenue',
    ADD COLUMN IF NOT EXISTS sales_goal_target NUMERIC(14,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS sales_goal_months JSONB NOT NULL DEFAULT '{}'::jsonb;

UPDATE pixels
SET sales_goal_months = jsonb_build_object(
    to_char(NOW() AT TIME ZONE 'America/Sao_Paulo', 'YYYY-MM'),
    sales_goal_target
)
WHERE sales_goal_target > 0
  AND sales_goal_months = '{}'::jsonb;
