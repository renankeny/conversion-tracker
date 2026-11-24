-- =====================================================================
-- 0070_pixel_monthly_sales_goals.sql
-- Metas mensais por período (YYYY-MM) em pixels
-- =====================================================================

ALTER TABLE pixels
    ADD COLUMN IF NOT EXISTS sales_goal_months JSONB NOT NULL DEFAULT '{}'::jsonb;

UPDATE pixels
SET sales_goal_months = jsonb_build_object(
    to_char(NOW() AT TIME ZONE 'America/Sao_Paulo', 'YYYY-MM'),
    sales_goal_target
)
WHERE sales_goal_target > 0
  AND sales_goal_months = '{}'::jsonb;
