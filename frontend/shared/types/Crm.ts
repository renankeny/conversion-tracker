import type { TrackedUser } from './TrackedUser'

export interface CrmStage {
  id: number
  account_id: number
  name: string
  color: string
  order_index: number
  trigger_event: string
  google_ads_account_id: number | null
  google_ads_action_id: string
  created_at?: string
  updated_at?: string
}

export interface CrmStageForm {
  id?: number
  name: string
  color: string
  trigger_event: string
  google_ads_account_id: number | null
  google_ads_action_id: string
}

export interface CrmTag {
  id: number
  account_id?: number
  name: string
  color: string
  created_at?: string
}

export interface CrmNote {
  id: number
  crm_lead_id: number
  content: string
  created_at: string
}

export interface CrmLead {
  id: number
  account_id: number
  tracked_user_id: number
  stage_id: number | null
  value: number
  has_notes?: boolean
  created_at: string
  updated_at: string
  tracked_user?: TrackedUser | null
  notes?: CrmNote[]
  tags?: CrmTag[]
}

export interface CrmBoardResponse {
  stages: CrmStage[]
  leads: CrmLead[]
}

export interface CrmOriginBadge {
  label: string
  class: 'origin-meta' | 'origin-google' | 'origin-tiktok' | 'origin-direct' | 'origin-other'
}

export type CrmOriginCategory = 'meta' | 'google' | 'tiktok' | 'direct' | 'other'

export interface CrmCreateLeadForm {
  first_name: string
  last_name: string
  email: string
  phone: string
  value: number
  stage_id: number | null
}

export interface CrmLeadEditState {
  first_name: string
  last_name: string
  email: string
  phone: string
  value: number
  stage_id: number | null
  notes: CrmNote[]
  tags: CrmTag[]
}

