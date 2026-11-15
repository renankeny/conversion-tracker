-- 0061_pixel_ga4_text_columns.sql
-- Altera ga4_measurement_id e ga4_api_secret em pixels para TEXT,
-- evitando estouro de VARCHAR(100) ao criptografar ga4_api_secret com AES-GCM em hex.

ALTER TABLE pixels
    ALTER COLUMN ga4_measurement_id TYPE TEXT,
    ALTER COLUMN ga4_api_secret TYPE TEXT;
