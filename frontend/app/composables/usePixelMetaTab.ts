import type { AdAccountSelectOption } from '~/components/BaseAdAccountSelect.vue'
import type { MetaAccount, MetaRemoteAdAccount, MetaRemotePixel, TrackerConfig } from '#shared/types'

const MANUAL_OPTION = '__manual__'

export interface MetaTabFormState {
  account_id: number | null
  pixel_ids: string[]
  access_tokens: string[]
  meta_account_id: number | null
  meta_ad_account_id: string
  tracker_config: TrackerConfig
}

const normalizeAct = (a: MetaRemoteAdAccount): string => {
  const raw = (a.id || a.account_id || '').trim()
  return !raw ? '' : raw.startsWith('act_') ? raw : `act_${raw}`
}

const accountStatusBadge = (acc: MetaAccount, actId: string): string | undefined => {
  if (!acc.enabled) return 'Pausada'
  if (acc.token_expired) return 'Token expirado'
  if (!acc.connected) return 'Pendente'
  return !actId ? 'Incompleta' : undefined
}

export const usePixelMetaTab = (props: { form: MetaTabFormState; isEditing: boolean }) => {
  const metaApi = useMetaAccounts()
  const metaAccounts = ref<MetaAccount[]>([])
  const remoteAccountsByMeta = reactive<Record<number, MetaRemoteAdAccount[]>>({})
  const oauthPicByMeta = reactive<Record<number, string>>({})
  const remotePixels = ref<MetaRemotePixel[]>([])
  const loadingAccounts = ref(false)
  const loadingPixels = ref(false)
  const manualRows = reactive<Record<number, boolean>>({})
  const userClearedAccount = ref(false)

  const metaAccountOptions = computed<AdAccountSelectOption[]>(() => {
    const opts: AdAccountSelectOption[] = [
      { value: '0', label: 'Nenhuma (usar apenas Access Token manual)', channel: 'none' },
    ]
    for (const acc of metaAccounts.value) {
      const remotes = remoteAccountsByMeta[acc.id] || []
      if (remotes.length) {
        for (const rem of remotes) {
          const actId = normalizeAct(rem)
          opts.push({
            value: `${acc.id}:${actId}`,
            label: rem.name || acc.ad_account_name || acc.name,
            sublabel: actId,
            avatarUrl: rem.picture_url || oauthPicByMeta[acc.id] || acc.ad_account_picture_url || acc.picture_url || '',
            channel: 'meta',
          })
        }
      } else {
        const actId = acc.ad_account_id || ''
        opts.push({
          value: `${acc.id}:${actId}`,
          label: acc.ad_account_name || acc.name,
          sublabel: actId || acc.oauth_name || acc.oauth_email || '',
          avatarUrl: acc.ad_account_picture_url || acc.picture_url || acc.oauth_picture_url || '',
          channel: 'meta',
          badge: accountStatusBadge(acc, actId),
        })
      }
    }
    return opts
  })

  const pixelDropdownOptions = computed(() => [
    ...remotePixels.value.map((p) => ({ value: p.id, label: `${p.name} (${p.id})` })),
    { value: MANUAL_OPTION, label: 'Digitar outro Pixel ID manualmente...' },
  ])

  const pixelDropdownValue = (idx: number): string => {
    const cur = (props.form.pixel_ids[idx] || '').trim()
    if (!cur) return ''
    return remotePixels.value.some((p) => p.id === cur) ? cur : MANUAL_OPTION
  }

  const onSelectPixel = (idx: number, val: string) => {
    if (val === MANUAL_OPTION) {
      manualRows[idx] = true
      return
    }
    manualRows[idx] = false
    props.form.pixel_ids[idx] = val
    if (idx === 0 || !props.form.tracker_config.meta_init_pixel_id) {
      props.form.tracker_config.meta_init_pixel_id = val
    }
  }

  const switchToList = (idx: number) => {
    manualRows[idx] = false
    if (remotePixels.value.length && !remotePixels.value.some((p) => p.id === props.form.pixel_ids[idx])) {
      props.form.pixel_ids[idx] = remotePixels.value[0]!.id
    }
  }

  const loadPixels = async (accId: number | null, adAccountId: string, replacePreviousAuto = false) => {
    const prevPixelIds = new Set(remotePixels.value.map((p) => p.id))
    remotePixels.value = []
    if (!accId || accId <= 0) return
    const acc = metaAccounts.value.find((a) => a.id === accId)
    if (!acc || !acc.connected) return

    loadingPixels.value = true
    try {
      const targetAct = adAccountId || acc.ad_account_id
      const pxRes = await metaApi.listPixels(acc.id, targetAct, false, props.form.account_id)
      remotePixels.value = pxRes.pixels || []

      for (let i = 0; i < props.form.pixel_ids.length; i++) {
        const cur = (props.form.pixel_ids[i] || '').trim()
        manualRows[i] = !!cur && !remotePixels.value.some((p) => p.id === cur)
      }

      const firstPx = remotePixels.value[0]
      if (firstPx) {
        const curFirst = (props.form.pixel_ids[0] || '').trim()
        if (!props.form.pixel_ids.length) {
          props.form.pixel_ids.push(firstPx.id)
          props.form.access_tokens.push('')
          manualRows[0] = false
        } else if (!curFirst || (replacePreviousAuto && prevPixelIds.has(curFirst))) {
          props.form.pixel_ids[0] = firstPx.id
          manualRows[0] = false
        }
        const curInit = props.form.tracker_config.meta_init_pixel_id
        if (!curInit || (replacePreviousAuto && prevPixelIds.has(curInit))) {
          props.form.tracker_config.meta_init_pixel_id = firstPx.id
        }
      } else if (!props.isEditing && !props.form.pixel_ids.length) {
        props.form.pixel_ids.push('')
        props.form.access_tokens.push('')
      }
    } catch {
      remotePixels.value = []
    } finally {
      loadingPixels.value = false
    }
  }

  const metaAccountSelectValue = computed<string>({
    get: () => {
      const id = props.form.meta_account_id
      if (!id || id <= 0) return '0'
      const acc = metaAccounts.value.find((a) => a.id === id)
      const firstRem = remoteAccountsByMeta[id]?.[0]
      const act = props.form.meta_ad_account_id || acc?.ad_account_id || (firstRem ? normalizeAct(firstRem) : '')
      return `${id}:${act}`
    },
    set: (val: string) => {
      if (!val || val === '0') {
        userClearedAccount.value = true
        props.form.meta_account_id = null
        props.form.meta_ad_account_id = ''
        remotePixels.value = []
        return
      }
      userClearedAccount.value = false
      const [idPart, actPart = ''] = String(val).split(':')
      const n = parseInt(idPart || '0', 10)
      props.form.meta_account_id = Number.isFinite(n) && n > 0 ? n : null
      props.form.meta_ad_account_id = actPart
      loadPixels(props.form.meta_account_id, actPart, true)
    },
  })

  const hydrateAccounts = async () => {
    loadingAccounts.value = true
    try {
      const targetAccId = props.form.account_id
      const all = await metaApi.listByAccount(targetAccId, true)
      metaAccounts.value = targetAccId && targetAccId > 0 ? all.filter((a) => a.account_id === targetAccId) : all

      if (props.form.meta_account_id && !metaAccounts.value.some((a) => a.id === props.form.meta_account_id)) {
        props.form.meta_account_id = null
        props.form.meta_ad_account_id = ''
        remotePixels.value = []
      }

      await Promise.all(metaAccounts.value.filter((a) => a.connected).map(async (acc) => {
        try {
          const res = await metaApi.listAdAccounts(acc.id, false, targetAccId)
          remoteAccountsByMeta[acc.id] = res.ad_accounts || []
          if (res.oauth_picture_url) oauthPicByMeta[acc.id] = res.oauth_picture_url
        } catch {
          remoteAccountsByMeta[acc.id] = []
        }
      }))

      if (!props.isEditing && !props.form.meta_account_id && !userClearedAccount.value) {
        const firstAcc = metaAccounts.value.find((a) => a.connected && !a.token_expired && a.enabled)
        if (firstAcc) {
          const firstRem = remoteAccountsByMeta[firstAcc.id]?.[0]
          const actId = firstAcc.ad_account_id || (firstRem ? normalizeAct(firstRem) : '')
          props.form.meta_account_id = firstAcc.id
          props.form.meta_ad_account_id = actId
          await loadPixels(firstAcc.id, actId, false)
        }
      } else if (props.form.meta_account_id) {
        const acc = metaAccounts.value.find((a) => a.id === props.form.meta_account_id)
        const firstRem = remoteAccountsByMeta[props.form.meta_account_id]?.[0]
        const actId = props.form.meta_ad_account_id || acc?.ad_account_id || (firstRem ? normalizeAct(firstRem) : '')
        if (actId && !props.form.meta_ad_account_id) props.form.meta_ad_account_id = actId
        await loadPixels(props.form.meta_account_id, actId, false)
      }
    } catch {
      // mantém estado local se offline
    } finally {
      loadingAccounts.value = false
    }
  }

  const addPixelRow = () => {
    const defaultId = !props.form.pixel_ids.length && remotePixels.value[0] ? remotePixels.value[0].id : ''
    props.form.pixel_ids.push(defaultId)
    props.form.access_tokens.push('')
  }

  const removePixelRow = (idx: number) => {
    delete manualRows[idx]
    props.form.pixel_ids.splice(idx, 1)
    props.form.access_tokens.splice(idx, 1)
  }

  watch(() => props.form.account_id, () => { hydrateAccounts() })
  onMounted(hydrateAccounts)

  return {
    remotePixels, loadingAccounts, loadingPixels, manualRows,
    metaAccountOptions, metaAccountSelectValue,
    pixelDropdownOptions, pixelDropdownValue,
    onSelectPixel, switchToList, addPixelRow, removePixelRow,
  }
}
