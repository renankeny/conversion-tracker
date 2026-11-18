-- =====================================================================
-- Migração 0066: Pré-configura eventos padrão de CTWA no funil CRM
-- - 1ª etapa padrão ("Novo Lead", order_index = 0) -> trigger_event_ctwa = 'LeadSubmitted'
--   (respeitando a configuração de 1ª ou 3ª mensagem na instância WhatsApp)
-- - Etapa padrão de vendas ("Venda Fechada") -> trigger_event_ctwa = 'Purchase'
-- Não sobrescreve etapas já customizadas pelo usuário.
-- =====================================================================

UPDATE crm_stages
SET trigger_event_ctwa = 'LeadSubmitted',
    updated_at = NOW()
WHERE order_index = 0
  AND LOWER(TRIM(name)) = 'novo lead'
  AND trigger_event = ''
  AND trigger_event_ctwa = '';

UPDATE crm_stages
SET trigger_event_ctwa = 'Purchase',
    updated_at = NOW()
WHERE (LOWER(TRIM(name)) = 'venda fechada' OR LOWER(trigger_event) = 'purchase')
  AND trigger_event_ctwa = '';
