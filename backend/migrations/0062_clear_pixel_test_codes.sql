-- Limpa códigos de teste legados salvos diretamente na tabela pixels.
-- Disparo com test_event_code ocorre apenas de forma avulsa no endpoint de teste,
-- garantindo que nenhum pixel em produção fique preso em modo teste.
UPDATE pixels
SET test_codes = '{}',
    tiktok_test_codes = '{}'
WHERE cardinality(test_codes) > 0
   OR cardinality(tiktok_test_codes) > 0;
