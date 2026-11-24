-- =====================================================================
-- 0072_pixel_sales_goal_backfill_fix.sql
-- Garantia idempotente pós-0071 para convergência de metas mensais
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
