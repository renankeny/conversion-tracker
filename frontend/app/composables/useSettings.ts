import type {
  AdminUser,
  AdminUsersListResponse,
  ChangePasswordInput,
  CreateAdminInput,
  GeoIPRefreshResponse,
  GeoIPStatus,
  MaintenanceStatusResponse,
  PruneResponse,
  ResetAccountResponse,
  UpdateRetentionInput,
} from '#shared/types'

export const useSettings = () => {
  const api = useAPI()

  // Admin users
  const listAdmins = (): Promise<AdminUsersListResponse> =>
    api.get<AdminUsersListResponse>('/admin/users')

  const createAdmin = (input: CreateAdminInput): Promise<AdminUser> =>
    api.post<AdminUser>('/admin/users', input)

  const deleteAdmin = (id: number): Promise<void> =>
    api.delete<void>(`/admin/users/${id}`, { body: { confirm: true } })

  const changeMyPassword = (input: ChangePasswordInput): Promise<{ ok: boolean }> =>
    api.post<{ ok: boolean }>('/admin/me/password', input)

  // GeoIP
  const geoipStatus = (): Promise<GeoIPStatus> =>
    api.get<GeoIPStatus>('/admin/geoip/status')

  const geoipRefresh = (): Promise<GeoIPRefreshResponse> =>
    api.post<GeoIPRefreshResponse>('/admin/geoip/refresh')

  // Maintenance
  const maintenanceStatus = (): Promise<MaintenanceStatusResponse> =>
    api.get<MaintenanceStatusResponse>('/admin/maintenance/status')

  const pruneNow = (
    sentEventsDays?: number,
    eventCountersDays?: number
  ): Promise<PruneResponse> => {
    const params: string[] = []
    if (sentEventsDays !== undefined) {
      params.push('sent_events_days=' + sentEventsDays)
    }
    if (eventCountersDays !== undefined) {
      params.push('event_counters_days=' + eventCountersDays)
    }
    const qs = params.length ? '?' + params.join('&') : ''
    return api.post<PruneResponse>('/admin/maintenance/prune' + qs)
  }

  // Atualiza override de retenção da conta. Passe null em qualquer campo
  // pra resetar pro default global.
  const updateRetention = (input: UpdateRetentionInput): Promise<void> =>
    api.put<void>('/admin/maintenance/retention', input)

  // Apaga TODOS os sent_events e event_counters da conta. Operação destrutiva.
  const resetAccount = (accountID?: number): Promise<ResetAccountResponse> =>
    api.post<ResetAccountResponse>('/admin/maintenance/reset', {
      confirm: true,
      account_id: accountID,
    })

  return {
    listAdmins,
    createAdmin,
    deleteAdmin,
    changeMyPassword,
    geoipStatus,
    geoipRefresh,
    maintenanceStatus,
    pruneNow,
    updateRetention,
    resetAccount,
  }
}
