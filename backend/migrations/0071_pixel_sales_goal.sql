-- 0071_pixel_sales_goal.sql
-- Meta de vendas configurável por Pixel (em faturamento ou em quantidade de vendas).

ALTER TABLE pixels
    ADD COLUMN IF NOT EXISTS sales_goal_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS sales_goal_type VARCHAR(16) NOT NULL DEFAULT 'revenue',
    ADD COLUMN IF NOT EXISTS sales_goal_target NUMERIC(14,2) NOT NULL DEFAULT 0;
