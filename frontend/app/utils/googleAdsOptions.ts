import type { AdAccountSelectOption } from '~/components/BaseAdAccountSelect.vue'
import type { GoogleAdsAccount, GoogleAdsRemoteCustomer } from '#shared/types'

export const stripGoogleCidDashes = (s: string): string => (s || '').replace(/\D/g, '')

export const formatGoogleCid = (raw: string): string => {
  const d = stripGoogleCidDashes(raw)
  return d.length === 10 ? `${d.slice(0, 3)}-${d.slice(3, 6)}-${d.slice(6)}` : raw
}

export const isGoogleOAuthConnected = (a: GoogleAdsAccount): boolean =>
  Boolean(a.oauth_connected || a.connected)

export const googleOAuthKeyFor = (a: GoogleAdsAccount): string => {
  const email = (a.oauth_email || '').trim().toLowerCase()
  return isGoogleOAuthConnected(a) && email ? `email:${email}` : `id:${a.id}`
}

export function buildGoogleOAuthOptions(
  accounts: GoogleAdsAccount[],
  oauthPictureByKey: Record<string, string>,
  pickRep: (key: string) => GoogleAdsAccount | null,
): AdAccountSelectOption[] {
  const items: AdAccountSelectOption[] = [
    { value: '0', label: 'Nenhuma (sem vínculo)', channel: 'none' },
  ]
  const seen = new Set<string>()
  for (const a of accounts) {
    const key = googleOAuthKeyFor(a)
    if (seen.has(key)) continue
    seen.add(key)
    const rep = pickRep(key) || a
    const ok = isGoogleOAuthConnected(rep)
    items.push({
      value: key,
      label: rep.oauth_email || rep.name,
      sublabel: ok ? 'Conta Google conectada via OAuth' : 'Aguardando conexão OAuth',
      avatarUrl: oauthPictureByKey[key] || rep.picture_url || rep.oauth_picture_url || '',
      channel: 'google_ads',
      badge: !rep.enabled ? 'Pausada' : (!ok ? 'Pendente' : undefined),
    })
  }
  return items
}

export function buildGoogleCustomerOptions(params: {
  remoteCustomers: GoogleAdsRemoteCustomer[]
  savedGroup: GoogleAdsAccount[]
  selectedCleanId: string
  selectedName: string
  avatarUrl: string
}): AdAccountSelectOption[] {
  const { remoteCustomers, savedGroup, selectedCleanId, selectedName, avatarUrl } = params
  const seen = new Set<string>()
  const list: AdAccountSelectOption[] = []

  for (const c of remoteCustomers) {
    const val = stripGoogleCidDashes(c.id)
    if (!val || seen.has(val)) continue
    seen.add(val)
    const parts = [c.formatted_id || formatGoogleCid(val)]
    if (c.currency_code) parts.push(c.currency_code)
    if (c.manager_customer_id) parts.push(`MCC ${formatGoogleCid(c.manager_customer_id)}`)
    list.push({
      value: val,
      label: c.name,
      sublabel: parts.join(' · '),
      avatarUrl,
      channel: 'google_ads',
      badge: c.is_manager ? 'MCC' : undefined,
    })
  }

  for (const saved of savedGroup) {
    const val = stripGoogleCidDashes(saved.customer_id)
    if (!val || seen.has(val)) continue
    seen.add(val)
    list.push({
      value: val,
      label: saved.customer_name || saved.name,
      sublabel: formatGoogleCid(val),
      avatarUrl,
      channel: 'google_ads',
    })
  }

  if (selectedCleanId && !seen.has(selectedCleanId)) {
    list.unshift({
      value: selectedCleanId,
      label: selectedName || 'Conta selecionada',
      sublabel: formatGoogleCid(selectedCleanId),
      avatarUrl,
      channel: 'google_ads',
    })
  }

  return list
}
