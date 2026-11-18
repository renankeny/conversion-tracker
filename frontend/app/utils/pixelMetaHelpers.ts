import type { AdAccountSelectOption } from '~/components/BaseAdAccountSelect.vue'
import type { MetaAccount, MetaRemoteAdAccount } from '#shared/types'

export const normalizeMetaAdAccountId = (a: MetaRemoteAdAccount): string => {
  const raw = (a.id || a.account_id || '').trim()
  return !raw ? '' : raw.startsWith('act_') ? raw : `act_${raw}`
}

export const metaAccountStatusBadge = (acc: MetaAccount, actId: string): string | undefined => {
  if (!acc.enabled) return 'Pausada'
  if (acc.token_expired) return 'Token expirado'
  if (!acc.connected) return 'Pendente'
  return !actId ? 'Incompleta' : undefined
}

export const shiftIndexedFlags = (map: Record<number, boolean>, removedIdx: number, newLen: number) => {
  for (let i = removedIdx; i < newLen; i++) {
    map[i] = Boolean(map[i + 1])
  }
  delete map[newLen]
}

export const buildMetaAccountSelectOptions = (
  metaAccounts: MetaAccount[],
  remoteAccountsByMeta: Record<number, MetaRemoteAdAccount[]>,
  oauthPicByMeta: Record<number, string>,
): AdAccountSelectOption[] => {
  const opts: AdAccountSelectOption[] = [
    { value: '0', label: 'Nenhuma (usar apenas Access Token manual)', channel: 'none' },
  ]
  for (const acc of metaAccounts) {
    const remotes = remoteAccountsByMeta[acc.id] || []
    if (remotes.length) {
      for (const rem of remotes) {
        const actId = normalizeMetaAdAccountId(rem)
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
        badge: metaAccountStatusBadge(acc, actId),
      })
    }
  }
  return opts
}
