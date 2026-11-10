-- =====================================================================
-- 0046_reclassify_traffic.sql
--
-- Reclassifica registros existentes em tracked_users que foram criados
-- antes da migration 0044 ou com valores padrão incorretos.
--
-- Exemplo: Meta External Ads Crawler
-- (meta-externalads/1.1 (+https://developers.facebook.com/docs/sharing/webmasters/crawler))
-- =====================================================================

-- 1. Crawlers de Plataformas de Anúncios e Redes Sociais (Meta, Google Ads, TikTok, etc.)
UPDATE tracked_users
SET 
    traffic_type = 'bot',
    traffic_score = -80,
    traffic_category = 'ad_verifier',
    traffic_rules = '[{"id":"ua_ad_verifier","label":"Crawler de Plataforma de Anúncios / Pré-visualização","impact":"automated","weight":-80,"description":"User-Agent característico de robô de inspeção de anúncios ou gerador de links (Meta, Google Ads, TikTok, etc.)."}]'::jsonb
WHERE 
    LOWER(client_user_agent) LIKE '%meta-external%'
    OR LOWER(client_user_agent) LIKE '%facebookexternalhit%'
    OR LOWER(client_user_agent) LIKE '%facebookcatalog%'
    OR LOWER(client_user_agent) LIKE '%facebot%'
    OR LOWER(client_user_agent) LIKE '%adsbot-google%'
    OR LOWER(client_user_agent) LIKE '%google-adwords-instant%'
    OR LOWER(client_user_agent) LIKE '%tiktokbot%'
    OR LOWER(client_user_agent) LIKE '%twitterbot%'
    OR LOWER(client_user_agent) LIKE '%telegrambot%'
    OR LOWER(client_user_agent) LIKE '%linkedinbot%';

-- 2. Indexadores de Busca (Googlebot, Bingbot, etc.)
UPDATE tracked_users
SET 
    traffic_type = 'bot',
    traffic_score = -75,
    traffic_category = 'search_engine',
    traffic_rules = '[{"id":"ua_search_bot","label":"Crawler de Motor de Busca","impact":"automated","weight":-75,"description":"User-Agent identificado como indexador de pesquisa web (Googlebot, Bingbot, etc.)."}]'::jsonb
WHERE 
    traffic_category = 'none' AND (
       LOWER(client_user_agent) LIKE '%googlebot%'
    OR LOWER(client_user_agent) LIKE '%bingbot%'
    OR LOWER(client_user_agent) LIKE '%yandexbot%'
    OR LOWER(client_user_agent) LIKE '%duckduckbot%'
    OR LOWER(client_user_agent) LIKE '%baiduspider%'
    );

-- 3. Scrapers e Ambientes Headless
UPDATE tracked_users
SET 
    traffic_type = 'bot',
    traffic_score = -90,
    traffic_category = 'headless_scraper',
    traffic_rules = '[{"id":"ua_headless_scraper","label":"Ambiente Headless / Automação / Scraper","impact":"automated","weight":-90,"description":"Identificado software de automação sem interface visual, biblioteca de raspagem ou bot de inteligência artificial."}]'::jsonb
WHERE 
    traffic_category = 'none' AND (
       LOWER(client_user_agent) LIKE '%headless%'
    OR LOWER(client_user_agent) LIKE '%playwright%'
    OR LOWER(client_user_agent) LIKE '%puppeteer%'
    OR LOWER(client_user_agent) LIKE '%selenium%'
    OR LOWER(client_user_agent) LIKE '%python%'
    OR LOWER(client_user_agent) LIKE '%curl%'
    OR LOWER(client_user_agent) LIKE '%wget%'
    OR LOWER(client_user_agent) LIKE '%scrapy%'
    OR LOWER(client_user_agent) LIKE '%bytespider%'
    OR LOWER(client_user_agent) LIKE '%gptbot%'
    OR LOWER(client_user_agent) LIKE '%claudebot%'
    );

-- 4. Assinaturas Genéricas de Robôs e Crawlers
UPDATE tracked_users
SET 
    traffic_type = 'bot',
    traffic_score = -60,
    traffic_category = 'generic_bot',
    traffic_rules = '[{"id":"ua_generic_bot","label":"Assinatura Genérica de Robô","impact":"automated","weight":-60,"description":"Cabeçalho User-Agent inclui termos explícitos de robô/rastreador web."}]'::jsonb
WHERE 
    traffic_category = 'none' AND (
       LOWER(client_user_agent) LIKE '%crawler%'
    OR LOWER(client_user_agent) LIKE '%spider%'
    OR LOWER(client_user_agent) LIKE '%bot%'
    );
