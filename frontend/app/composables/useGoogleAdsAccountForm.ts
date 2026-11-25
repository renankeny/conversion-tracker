import { errorMessage } from '~/utils/errorMessage'
import type {
  GoogleAdsAccount,
  GoogleAdsAccountInput,
  GoogleAdsConversionAction,
  GoogleAdsRemoteConversionAction,
  GoogleAdsRemoteCustomer,
} from '#shared/types'

export const stripGadsDashes = (s: string) => (s || '').replace(/\D/g, '')

export const isGadsUploadType = (tp?: string) => {
  const u = (tp || '').toUpperCase().trim()
  return u === 'UPLOAD_CLICKS' || u === 'UPLOAD_CALLS' || u === 'STORE_SALES' || u === 'STORE_SALES_DIRECT_UPLOAD'
}

export const useGoogleAdsAccountForm = (
  accountRef: Ref<GoogleAdsAccount | null | undefined>,
  emitSubmit: (input: GoogleAdsAccountInput) => void,
  emitConnected: (account: GoogleAdsAccount) => void,
  emitRefresh: () => void,
) => {
  const googleAdsApi = useGoogleAdsAccounts()
  const toast = useToast()

  const defaultConversionRow = (): GoogleAdsConversionAction => ({
    event_type: 'Purchase', conversion_action_id: '', name: '', enabled: true,
  })

  const form = reactive({
    name: '',
    customer_id: '',
    customer_name: '',
    login_customer_id: '',
    conversion_actions: [defaultConversionRow()] as GoogleAdsConversionAction[],
    enabled: true,
  })

  const errors = reactive({ name: '' })
  const oauthPictureUrl = ref('')
  const remoteCustomers = ref<GoogleAdsRemoteCustomer[]>([])
  const loadingCustomers = ref(false)
  const remoteActions = ref<GoogleAdsRemoteConversionAction[]>([])
  const loadingActions = ref(false)
  const creatingAction = ref(false)
  const lastLoadedCustomerDigits = ref('')

  const isAccConnected = (acc?: GoogleAdsAccount | null) =>
    Boolean(acc?.oauth_connected || acc?.connected || acc?.oauth_email)

  const oauthConnected = computed(() => isAccConnected(accountRef.value))
  const canLoadRemoteActions = computed(
    () => oauthConnected.value && Boolean(stripGadsDashes(form.customer_id)),
  )

  const syncSavedActionMetadata = () => {
    for (const row of form.conversion_actions) {
      const remote = remoteActions.value.find((r) => r.id === row.conversion_action_id)
      if (remote) {
        if (!row.name) row.name = remote.name
        row.action_type = remote.type
        row.owner_customer_id = remote.owner_customer_id || ''
      }
    }
  }

  const loadRemoteActions = async (forceRefresh = false, targetAccount?: GoogleAdsAccount) => {
    const acc = targetAccount ?? accountRef.value
    const cleanCid = stripGadsDashes(form.customer_id)
    if (!acc || !cleanCid) return
    loadingActions.value = true
    try {
      const res = await googleAdsApi.listConversionActions(acc.id, form.customer_id, form.login_customer_id, forceRefresh)
      remoteActions.value = res.conversion_actions || []
      lastLoadedCustomerDigits.value = cleanCid
      syncSavedActionMetadata()
    } catch (e: unknown) {
      toast.error('Erro ao buscar Ações de Conversão', errorMessage(e, 'Tente novamente'))
    } finally {
      loadingActions.value = false
    }
  }

  const onCustomerSelected = async (customer: GoogleAdsRemoteCustomer) => {
    const nextDigits = stripGadsDashes(customer.id)
    if (lastLoadedCustomerDigits.value && lastLoadedCustomerDigits.value !== nextDigits) {
      form.conversion_actions = [defaultConversionRow()]
    }
    await loadRemoteActions(true)
  }

  const onCreateRemoteAction = async (row: GoogleAdsConversionAction) => {
    if (!accountRef.value?.id || !stripGadsDashes(form.customer_id)) return
    creatingAction.value = true
    try {
      const cur = remoteCustomers.value.find((c) => stripGadsDashes(c.id) === stripGadsDashes(form.customer_id))?.currency_code || 'BRL'
      const res = await googleAdsApi.createConversionAction(accountRef.value.id, {
        customer_id: form.customer_id,
        login_customer_id: form.login_customer_id || undefined,
        event_type: row.event_type || 'Purchase',
        currency: cur,
      })
      const created = res.conversion_action
      if (created?.id) {
        remoteActions.value = [created, ...remoteActions.value.filter((r) => r.id !== created.id)]
        row.conversion_action_id = created.id
        row.name = created.name
        row.action_type = created.type
        row.owner_customer_id = created.owner_customer_id || ''
        row.enabled = true
        toast.success('Conversão criada no Google Ads', `"${created.name}" criada como UPLOAD_CLICKS e vinculada.`)
      }
    } catch (e: unknown) {
      toast.error('Falha ao criar conversão', errorMessage(e, 'Verifique as permissões da conta Google Ads'))
    } finally {
      creatingAction.value = false
    }
  }

  const loadRemoteCustomers = async (forceRefresh = false, targetAccount?: GoogleAdsAccount) => {
    const acc = targetAccount ?? accountRef.value
    if (!acc || !isAccConnected(acc)) return
    loadingCustomers.value = true
    try {
      const res = await googleAdsApi.listCustomers(acc.id, forceRefresh)
      remoteCustomers.value = res.customers || []
      if (res.oauth_picture_url) oauthPictureUrl.value = res.oauth_picture_url
      if (res.warning && !remoteCustomers.value.length) toast.warning('Aviso do Google Ads', res.warning)

      const nonManager = remoteCustomers.value.filter((c) => !c.is_manager)
      const autoPick = nonManager.length === 1 ? nonManager[0] : (remoteCustomers.value.length === 1 ? remoteCustomers.value[0] : undefined)
      if (!form.customer_id && autoPick) {
        form.customer_id = autoPick.formatted_id || autoPick.id
        form.customer_name = autoPick.name || ''
        form.login_customer_id = autoPick.manager_customer_id || ''
        if (!form.name.trim() || form.name === 'Google Ads') form.name = autoPick.name
      } else if (form.customer_id && !form.customer_name) {
        const found = remoteCustomers.value.find((c) => stripGadsDashes(c.id) === stripGadsDashes(form.customer_id))
        if (found?.name) form.customer_name = found.name
      }
      if (form.customer_id) await loadRemoteActions(forceRefresh, acc)
    } catch (e: unknown) {
      toast.error('Erro ao buscar contas Google Ads', errorMessage(e, 'Tente novamente'))
    } finally {
      loadingCustomers.value = false
    }
  }

  const fillFromAccount = (a: GoogleAdsAccount) => {
    Object.assign(form, {
      name: a.name, customer_id: a.customer_id || '', customer_name: a.customer_name || '',
      login_customer_id: a.login_customer_id || '', enabled: a.enabled,
    })
    lastLoadedCustomerDigits.value = stripGadsDashes(a.customer_id || '')
    oauthPictureUrl.value = a.picture_url || a.oauth_picture_url || ''
    const mapped = (a.conversion_actions || []).map((ca) => ({
      event_type: ca.event_type, conversion_action_id: ca.conversion_action_id,
      name: ca.name || '', action_type: ca.action_type || '',
      owner_customer_id: ca.owner_customer_id || '', enabled: ca.enabled !== false,
    }))
    form.conversion_actions = mapped.length ? mapped : [defaultConversionRow()]
  }

  watch(accountRef, async (acc) => {
    remoteCustomers.value = []
    remoteActions.value = []
    if (acc) {
      fillFromAccount(acc)
      if (isAccConnected(acc)) await loadRemoteCustomers(false, acc)
    } else {
      oauthPictureUrl.value = ''
      lastLoadedCustomerDigits.value = ''
      Object.assign(form, {
        name: '', customer_id: '', customer_name: '', login_customer_id: '',
        conversion_actions: [defaultConversionRow()], enabled: true,
      })
    }
  }, { immediate: true })

  const onOAuthConnected = async (updated: GoogleAdsAccount) => {
    if (form.name.trim() && form.name !== 'Google Ads') updated = { ...updated, name: form.name.trim() }
    emitConnected(updated)
    await loadRemoteCustomers(true, updated)
  }

  const onOAuthDisconnected = () => {
    remoteCustomers.value = []
    remoteActions.value = []
    lastLoadedCustomerDigits.value = ''
    Object.assign(form, { customer_id: '', customer_name: '', login_customer_id: '', conversion_actions: [defaultConversionRow()] })
    emitRefresh()
  }

  const buildValidatedInput = (): boolean => {
    errors.name = ''
    if (!form.name.trim()) { errors.name = 'Nome é obrigatório'; return false }
    const seen = new Set<string>()
    const dedupedActions = form.conversion_actions
      .map((ca) => ({
        event_type: ca.event_type, conversion_action_id: ca.conversion_action_id.trim(),
        name: ca.name.trim(), action_type: ca.action_type || undefined,
        owner_customer_id: ca.owner_customer_id || undefined, enabled: ca.enabled,
      }))
      .filter((ca) => {
        if (!ca.event_type || !ca.conversion_action_id) return false
        const key = `${ca.event_type}:${ca.conversion_action_id}`
        if (seen.has(key)) return false
        seen.add(key)
        return true
      })
    emitSubmit({
      name: form.name.trim(), customer_id: form.customer_id.trim(), customer_name: form.customer_name.trim(),
      login_customer_id: form.login_customer_id.trim(), conversion_actions: dedupedActions, enabled: form.enabled,
    })
    return true
  }

  return {
    form, errors, oauthPictureUrl, remoteCustomers, loadingCustomers,
    remoteActions, loadingActions, creatingAction, oauthConnected, canLoadRemoteActions,
    loadRemoteCustomers, loadRemoteActions, onCustomerSelected, onCreateRemoteAction,
    onOAuthConnected, onOAuthDisconnected, buildValidatedInput,
  }
}
