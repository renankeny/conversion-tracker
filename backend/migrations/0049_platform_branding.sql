-- =====================================================================
-- 0049_platform_branding.sql
-- Identidade visual Whitelabel global da instância (nome, tagline, logos,
-- favicon e paleta de cores Dark/Light).
-- =====================================================================

CREATE TABLE IF NOT EXISTS platform_branding (
    id                SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    platform_name     TEXT NOT NULL DEFAULT 'Sua Plataforma',
    tagline           TEXT NOT NULL DEFAULT 'Intelligence & Attribution',
    footer_text       TEXT NOT NULL DEFAULT '',
    logo_display_mode TEXT NOT NULL DEFAULT 'icon_text', -- 'icon_text' | 'logo_only' | 'logo_and_text'
    logo_dark_url     TEXT NOT NULL DEFAULT '',
    logo_light_url    TEXT NOT NULL DEFAULT '',
    favicon_url       TEXT NOT NULL DEFAULT '',
    theme_config      JSONB NOT NULL DEFAULT '{
        "preset": "emerald",
        "dark": {
            "brand": "#A2F07C",
            "surface_1": "#29292E",
            "surface_2": "#323238",
            "surface_deep": "#202024"
        },
        "light": {
            "brand": "#0B4B4D",
            "surface_1": "#FFFFFF",
            "surface_2": "#FFFFFF",
            "surface_deep": "#F1F5F9"
        }
    }'::jsonb,
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO platform_branding (id, platform_name, tagline)
VALUES (1, 'Sua Plataforma', 'Intelligence & Attribution')
ON CONFLICT (id) DO NOTHING;

