import { computed } from 'vue'
import type {
  Account,
  GoogleAdsAccount,
  MetaAccount,
  Pixel,
  WhatsAppInstance,
} from '#shared/types'

export type OnboardingPlatform = 'meta' | 'google' | 'both'
export type OnboardingCampaignType = 'messages' | 'site' | 'both'
export type WizardStepKey = 'setup' | 'meta' | 'google' | 'pixel' | 'whatsapp' | 'done'

export interface WizardStepMeta {
  key: WizardStepKey
  label: string
}

export interface ReusableOAuthSource {
  id: number
  label: string
  sublabel: string
  avatarUrl: string
}

export const slugifyClientName = (val: string): string =>
  val
    .toLowerCase()
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')

export const useClientOnboardingWizard = () => {
  const isOpen = useState<boolean>('onboarding_wizard_open', () => false)
  const stepIndex = useState<number>('onboarding_wizard_step_idx', () => 0)

  const accountMode = useState<'new' | 'existing'>('onboarding_account_mode', () => 'new')
  const selectedExistingAccountId = useState<number>('onboarding_existing_account_id', () => 0)
  const clientName = useState<string>('onboarding_client_name', () => '')

  const platform = useState<OnboardingPlatform>('onboarding_platform', () => 'meta')
  const campaignType = useState<OnboardingCampaignType>('onboarding_campaign_type', () => 'messages')

  const createdAccount = useState<Account | null>('onboarding_created_account', () => null)
  const metaAccount = useState<MetaAccount | null>('onboarding_meta_account', () => null)
  const googleAccount = useState<GoogleAdsAccount | null>('onboarding_google_account', () => null)
  const createdPixel = useState<Pixel | null>('onboarding_created_pixel', () => null)
  const createdInstance = useState<WhatsAppInstance | null>('onboarding_created_instance', () => null)

  const skippedMetaOAuth = useState<boolean>('onboarding_skipped_meta', () => false)
  const skippedGoogleOAuth = useState<boolean>('onboarding_skipped_google', () => false)
  const reusableMetaSources = useState<ReusableOAuthSource[]>('onboarding_reuse_meta', () => [])
  const reusableGoogleSources = useState<ReusableOAuthSource[]>('onboarding_reuse_google', () => [])

  const steps = computed<WizardStepMeta[]>(() => {
    const list: WizardStepMeta[] = [{ key: 'setup', label: 'Cliente' }]
    if (platform.value === 'meta' || platform.value === 'both') {
      list.push({ key: 'meta', label: 'Meta Ads' })
    }
    if (platform.value === 'google' || platform.value === 'both') {
      list.push({ key: 'google', label: 'Google Ads' })
    }
    list.push({ key: 'pixel', label: 'Pixel' })
    if (campaignType.value === 'messages' || campaignType.value === 'both') {
      list.push({ key: 'whatsapp', label: 'WhatsApp' })
    }
    list.push({ key: 'done', label: 'Concluído' })
    return list
  })

  const currentStep = computed<WizardStepMeta>(() => {
    return steps.value[stepIndex.value] || steps.value[0]!
  })

  const includesMeta = computed(() => platform.value === 'meta' || platform.value === 'both')
  const includesGoogle = computed(() => platform.value === 'google' || platform.value === 'both')
  const includesSite = computed(() => campaignType.value === 'site' || campaignType.value === 'both')
  const includesWhatsApp = computed(() => campaignType.value === 'messages' || campaignType.value === 'both')
  const isMessagesOnly = computed(() => campaignType.value === 'messages')

  const captureReusableOAuthSources = async () => {
    const metaApi = useMetaAccounts()
    const gadsApi = useGoogleAdsAccounts()
    try {
      const [mList, gList] = await Promise.all([
        metaApi.list(false).catch(() => []),
        gadsApi.list(false).catch(() => []),
      ])
      const mSeen = new Set<string>()
      reusableMetaSources.value = (mList || [])
        .filter((a) => a.connected && !a.token_expired)
        .filter((a) => {
          const k = (a.oauth_user_id || a.oauth_email || a.oauth_name || String(a.id)).toLowerCase()
          if (mSeen.has(k)) return false
          mSeen.add(k)
          return true
        })
        .map((a) => ({
          id: a.id,
          label: a.oauth_name || a.oauth_email || a.name,
          sublabel: a.oauth_email || a.ad_account_name || 'Perfil Meta já conectado',
          avatarUrl: a.oauth_picture_url || a.picture_url || a.ad_account_picture_url || '',
        }))

      const gSeen = new Set<string>()
      reusableGoogleSources.value = (gList || [])
        .filter((a) => a.connected || a.oauth_connected)
        .filter((a) => {
          const k = (a.oauth_email || String(a.id)).toLowerCase()
          if (gSeen.has(k)) return false
          gSeen.add(k)
          return true
        })
        .map((a) => ({
          id: a.id,
          label: a.oauth_email || a.name,
          sublabel: a.customer_name || 'Conta Google já conectada',
          avatarUrl: a.picture_url || a.oauth_picture_url || '',
        }))
    } catch {
      // silencioso: o usuário ainda pode conectar normalmente via OAuth
    }
  }

  const resetWizard = () => {
    stepIndex.value = 0
    accountMode.value = 'new'
    selectedExistingAccountId.value = 0
    clientName.value = ''
    platform.value = 'meta'
    campaignType.value = 'messages'
    createdAccount.value = null
    metaAccount.value = null
    googleAccount.value = null
    createdPixel.value = null
    createdInstance.value = null
    skippedMetaOAuth.value = false
    skippedGoogleOAuth.value = false
  }

  const openWizard = () => {
    resetWizard()
    isOpen.value = true
    void captureReusableOAuthSources()
  }

  const closeWizard = () => {
    isOpen.value = false
  }

  const nextStep = () => {
    if (stepIndex.value < steps.value.length - 1) {
      stepIndex.value++
    }
  }

  const prevStep = () => {
    if (stepIndex.value > 0) {
      stepIndex.value--
    }
  }

  return {
    isOpen,
    stepIndex,
    steps,
    currentStep,
    accountMode,
    selectedExistingAccountId,
    clientName,
    platform,
    campaignType,
    createdAccount,
    metaAccount,
    googleAccount,
    createdPixel,
    createdInstance,
    skippedMetaOAuth,
    skippedGoogleOAuth,
    reusableMetaSources,
    reusableGoogleSources,
    includesMeta,
    includesGoogle,
    includesSite,
    includesWhatsApp,
    isMessagesOnly,
    openWizard,
    closeWizard,
    resetWizard,
    nextStep,
    prevStep,
  }
}
