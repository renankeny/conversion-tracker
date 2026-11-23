import { CORE_API_MODULES } from './modules_core'
import { WA_USERS_API_MODULES } from './modules_wa_users'
import { GROWTH_API_MODULES } from './modules_growth'
import type { ApiModuleGroup } from './types'

export * from './types'

export const API_MODULES: ApiModuleGroup[] = [
  ...CORE_API_MODULES,
  ...WA_USERS_API_MODULES,
  ...GROWTH_API_MODULES,
]

export interface McpToolCategory {
  title: string
  description: string
  tools: string[]
}

export const MCP_TOOL_CATEGORIES: McpToolCategory[] = [
  {
    title: 'WhatsApp e IA Comercial',
    description: 'Lê conversas, ouve transcrições de áudio/PIX, dispara a IA e converte chats em vendas.',
    tools: [
      'list_whatsapp_instances',
      'list_whatsapp_chats',
      'get_whatsapp_chat_messages',
      'get_whatsapp_chat_by_lead',
      'convert_whatsapp_chat',
      'run_whatsapp_ai_now',
      'get_whatsapp_stats',
    ],
  },
  {
    title: 'CRM e Funil 360',
    description: 'Consulta etapas, cria leads, move cards entre colunas, registra notas e aplica tags.',
    tools: [
      'list_crm_stages',
      'get_crm_board',
      'get_crm_lead_details',
      'upsert_crm_lead',
      'move_crm_lead',
      'update_crm_lead',
      'add_crm_lead_note',
      'assign_crm_lead_tag',
    ],
  },
  {
    title: 'Eventos, Vendas e Jornada',
    description: 'Envia conversões, audita disparos, confirma vendas pendentes e reconstrói a jornada completa.',
    tools: [
      'send_conversion_event',
      'list_sent_events',
      'get_event_by_id',
      'retry_failed_events',
      'list_pending_sales',
      'confirm_pending_sale',
      'lookup_tracked_user',
      'get_tracked_user_journey',
    ],
  },
  {
    title: 'ROAS Meta/Google e Links na Bio',
    description: 'Analisa faturamento, criativos campeões, palavras-chave e cria links rastreáveis.',
    tools: [
      'whoami_workspace',
      'list_pixels',
      'create_trackable_link',
      'get_dashboard_stats',
      'get_meta_ads_insights',
      'get_meta_creative_ranking',
      'get_google_ads_insights',
      'get_google_keyword_ranking',
    ],
  },
]
