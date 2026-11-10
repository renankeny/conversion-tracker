-- Migration: 0022_admin_users_preferences
-- Description: Adiciona coluna de preferências em JSON para admins (layout da dashboard, etc)

ALTER TABLE admin_users 
ADD COLUMN IF NOT EXISTS preferences JSONB NOT NULL DEFAULT '{}'::jsonb;
