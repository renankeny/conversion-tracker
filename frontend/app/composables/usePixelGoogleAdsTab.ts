import type { AdAccountValue } from '~/components/BaseAdAccountSelect.vue'
import type { GoogleAdsAccount, GoogleAdsRemoteCustomer } from '#shared/types'
import {
  stripGoogleCidDashes,
  formatGoogleCid,
  isGoogleOAuthConnected,
  googleOAuthKeyFor,
  buildGoogleOAuthOptions,
  buildGoogleCustomerOptions,
} from '~/utils/googleAdsOptions'

export interface GoogleAdsTabFormState {
  account_id: number | null
  google_ads_account_id: number | null
  google_ads_customer_id: string
  google_ads_customer_name: string
  google_ads_login_customer_id: string
}

export const usePixelGoogleAdsTab = (props: { form: GoogleAdsTabFormState }) => {
  const googleAdsApi = useGoogleAdsAccounts()
  const googleAdsAccounts = ref<GoogleAdsAccount[]>([])
  const remoteCustomers = ref<GoogleAdsRemoteCustomer[]>([])
  const oauthPictureByKey = reactive<Record<string, string>>({})
  const loadingAccounts = ref(false)
  const loadingCustomers = ref(false)
  const selectedOAuthKey = ref('0')
  let skipAccountWatch = false

  const clearCustomerFields = () => {
    props.form.google_ads_customer_id = ''
    props.form.google_ads_customer_name = ''
    props.form.google_ads_login_customer_id = ''
  }

  const groupAccounts = (key: string) => googleAdsAccounts.value.filter((a) => googleOAuthKeyFor(a) === key)

  const pickRepresentative = (key: string): GoogleAdsAccount | null => {
    const group = groupAccounts(key)
    return (
      group.find((a) => a.id === props.form.google_ads_account_id && a.enabled && isGoogleOAuthConnected(a)) ||
      group.find((a) => a.enabled && isGoogleOAuthConnected(a)) ||
      group.find((a) => isGoogleOAuthConnected(a)) ||
      group[0] ||
      null
    )
  }

  const activeOAuthAccount = computed(() =>
    selectedOAuthKey.value !== '0' ? pickRepresentative(selectedOAuthKey.value) : null,
  )

  const oauthAccountOptions = computed(() =>
    buildGoogleOAuthOptions(googleAdsAccounts.value, oauthPictureByKey, pickRepresentative),
  )

  const findMatchingSavedAccount = (cleanCid: string): GoogleAdsAccount | null => {
    if (!cleanCid || selectedOAuthKey.value === '0') return null
    const matches = groupAccounts(selectedOAuthKey.value).filter(
      (a) => stripGoogleCidDashes(a.customer_id) === cleanCid && isGoogleOAuthConnected(a),
    )
    return matches.find((a) => a.enabled && a.ready) || matches.find((a) => a.enabled) || matches[0] || null
  }

  const selectedCustomerCleanId = computed(() => stripGoogleCidDashes(props.form.google_ads_customer_id))
  const hasOfflineConversions = computed(() => Boolean(findMatchingSavedAccount(selectedCustomerCleanId.value)?.ready))

  const isSelectedManager = computed(() => {
    const cur = selectedCustomerCleanId.value
    return Boolean(cur && remoteCustomers.value.find((c) => stripGoogleCidDashes(c.id) === cur)?.is_manager)
  })

  const currentAvatar = computed(() => {
    const rep = activeOAuthAccount.value
    return oauthPictureByKey[selectedOAuthKey.value] || rep?.picture_url || rep?.oauth_picture_url || ''
  })

  const customerOptions = computed(() =>
    buildGoogleCustomerOptions({
      remoteCustomers: remoteCustomers.value,
      savedGroup: groupAccounts(selectedOAuthKey.value),
      selectedCleanId: selectedCustomerCleanId.value,
      selectedName: props.form.google_ads_customer_name,
      avatarUrl: currentAvatar.value,
    }),
  )

  const applyCustomerChoice = (cleanId: string) => {
    const rem = remoteCustomers.value.find((c) => stripGoogleCidDashes(c.id) === cleanId)
    const saved = findMatchingSavedAccount(cleanId)
    const rep = activeOAuthAccount.value
    skipAccountWatch = true
    props.form.google_ads_customer_id = rem?.formatted_id || formatGoogleCid(cleanId)
    props.form.google_ads_customer_name = rem?.name || saved?.customer_name || saved?.name || ''
    props.form.google_ads_login_customer_id = rem
      ? (rem.manager_customer_id ? formatGoogleCid(rem.manager_customer_id) : '')
      : (saved?.login_customer_id || '')
    props.form.google_ads_account_id = saved ? saved.id : (rep?.id || props.form.google_ads_account_id)
    void nextTick(() => { skipAccountWatch = false })
  }

  const loadCustomers = async (forceRefresh = false, autoPickSingle = false) => {
    const rep = activeOAuthAccount.value
    if (!rep || !isGoogleOAuthConnected(rep)) {
      remoteCustomers.value = []
      return
    }
    loadingCustomers.value = true
    try {
      const res = await googleAdsApi.listCustomers(rep.id, forceRefresh, props.form.account_id)
      remoteCustomers.value = res.customers || []
      if (res.oauth_picture_url) oauthPictureByKey[selectedOAuthKey.value] = res.oauth_picture_url
      const cur = selectedCustomerCleanId.value
      if (cur) {
        const found = remoteCustomers.value.find((c) => stripGoogleCidDashes(c.id) === cur)
        if (found) {
          if (!props.form.google_ads_customer_name) props.form.google_ads_customer_name = found.name || ''
          if (!props.form.google_ads_login_customer_id && found.manager_customer_id) {
            props.form.google_ads_login_customer_id = formatGoogleCid(found.manager_customer_id)
          }
        }
      } else if (autoPickSingle) {
        const nonManagers = remoteCustomers.value.filter((c) => !c.is_manager)
        const onlyOne = nonManagers.length === 1 ? nonManagers[0] : (remoteCustomers.value.length === 1 ? remoteCustomers.value[0] : null)
        if (onlyOne) applyCustomerChoice(stripGoogleCidDashes(onlyOne.id))
      }
    } catch {
      remoteCustomers.value = []
    } finally {
      loadingCustomers.value = false
    }
  }

  const onSelectOAuthAccount = (val: AdAccountValue) => {
    const nextKey = String(val || '0')
    if (nextKey === '0') {
      skipAccountWatch = true
      selectedOAuthKey.value = '0'
      props.form.google_ads_account_id = 0
      clearCustomerFields()
      remoteCustomers.value = []
      void nextTick(() => { skipAccountWatch = false })
      return
    }
    if (nextKey === selectedOAuthKey.value) return
    skipAccountWatch = true
    selectedOAuthKey.value = nextKey
    const rep = pickRepresentative(nextKey)
    props.form.google_ads_account_id = rep ? rep.id : 0
    clearCustomerFields()
    void nextTick(() => { skipAccountWatch = false })
    void loadCustomers(false, true)
  }

  const onSelectCustomer = (val: AdAccountValue) => {
    const cleanId = stripGoogleCidDashes(String(val || ''))
    if (cleanId) applyCustomerChoice(cleanId)
  }

  const syncFromFormState = async () => {
    const id = props.form.google_ads_account_id
    if (!id || id <= 0) {
      selectedOAuthKey.value = '0'
      remoteCustomers.value = []
      return
    }
    const linked = googleAdsAccounts.value.find((a) => a.id === id)
    if (!linked) {
      selectedOAuthKey.value = '0'
      props.form.google_ads_account_id = 0
      clearCustomerFields()
      remoteCustomers.value = []
      return
    }
    selectedOAuthKey.value = googleOAuthKeyFor(linked)
    if (!props.form.google_ads_customer_id && linked.customer_id) {
      props.form.google_ads_customer_id = formatGoogleCid(linked.customer_id)
      props.form.google_ads_customer_name = linked.customer_name || linked.name || ''
      props.form.google_ads_login_customer_id = linked.login_customer_id || ''
    }
    await loadCustomers(false, false)
  }

  const hydrateAccounts = async () => {
    loadingAccounts.value = true
    try {
      const targetAccId = props.form.account_id
      const all = await googleAdsApi.listByAccount(targetAccId, true)
      googleAdsAccounts.value = targetAccId && targetAccId > 0 ? all.filter((a) => a.account_id === targetAccId) : all
      await syncFromFormState()
    } catch {
      // mantém estado local caso offline
    } finally {
      loadingAccounts.value = false
    }
  }

  watch(() => props.form.account_id, () => { void hydrateAccounts() })
  watch(() => props.form.google_ads_account_id, () => {
    if (!skipAccountWatch && googleAdsAccounts.value.length) void syncFromFormState()
  })
  onMounted(() => { void hydrateAccounts() })

  return {
    googleAdsAccounts, loadingAccounts, loadingCustomers, selectedOAuthKey,
    selectedCustomerCleanId, oauthAccountOptions, customerOptions, activeOAuthAccount,
    isSelectedManager, hasOfflineConversions, onSelectOAuthAccount, onSelectCustomer, loadCustomers,
  }
}
