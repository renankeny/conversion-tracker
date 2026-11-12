import { ref, reactive, watch, type Ref, type ComputedRef } from 'vue'
import type { TrackedUser, TrackedUserUpdateInput } from '#shared/types'

const decodeUTM = (val: string | null | undefined): string => {
  if (!val) return ''
  try {
    return decodeURIComponent(val.replace(/\+/g, ' '))
  } catch (e) {
    return val.replace(/\+/g, ' ')
  }
}

export const useUserEditForm = (
  userRef: Ref<TrackedUser | null> | ComputedRef<TrackedUser | null>,
  onSavedCallback?: (updated: TrackedUser) => void
) => {
  const trackedUsers = useTrackedUsers()
  const toast = useToast()

  const editing = ref(false)
  const saving = ref(false)

  const form = reactive<Required<TrackedUserUpdateInput>>({
    id_wpp: '',
    pixel_name: '',
    fbp: '',
    fbc: '',
    ctwa_clid: '',
    gclid: '',
    wbraid: '',
    gbraid: '',
    ttclid: '',
    ttp: '',
    ga4_client_id: '',
    ga4_session_id: '',
    client_ip: '',
    client_user_agent: '',
    country: '',
    state: '',
    city: '',
    postal_code: '',
    first_name: '',
    last_name: '',
    email: '',
    phone: '',
    url: '',
    first_url: '',
    first_utm_source: '',
    first_utm_medium: '',
    first_utm_campaign: '',
    first_utm_content: '',
    first_utm_term: '',
    last_utm_source: '',
    last_utm_medium: '',
    last_utm_campaign: '',
    last_utm_content: '',
    last_utm_term: '',
  })

  const resetForm = (u: TrackedUser | null) => {
    if (!u) return
    const fields: (keyof TrackedUserUpdateInput)[] = [
      'id_wpp', 'pixel_name', 'fbp', 'fbc', 'ctwa_clid', 'gclid',
      'wbraid', 'gbraid', 'ttclid', 'ttp', 'ga4_client_id', 'ga4_session_id',
      'client_ip', 'client_user_agent',
      'country', 'state', 'city', 'postal_code',
      'first_name', 'last_name', 'email', 'phone', 'url', 'first_url',
      'first_utm_source', 'first_utm_medium', 'first_utm_campaign',
      'first_utm_content', 'first_utm_term',
      'last_utm_source', 'last_utm_medium', 'last_utm_campaign',
      'last_utm_content', 'last_utm_term',
    ]
    for (const f of fields) {
      let val = (u as any)[f] ?? ''
      if (typeof val === 'string' && f.includes('utm_')) {
        val = decodeUTM(val)
      }
      ;(form as any)[f] = val
    }
  }

  watch(
    () => userRef.value,
    (u) => {
      editing.value = false
      resetForm(u)
    },
    { immediate: true }
  )

  const startEdit = () => {
    resetForm(userRef.value)
    editing.value = true
  }

  const cancelEdit = () => {
    resetForm(userRef.value)
    editing.value = false
  }

  const onSave = async () => {
    const user = userRef.value
    if (!user) return
    saving.value = true
    try {
      const updated = await trackedUsers.update(user.id, { ...form })
      toast.success('Sucesso', 'Usuário atualizado com sucesso')
      editing.value = false
      if (onSavedCallback) onSavedCallback(updated)
      return updated
    } catch (error: any) {
      toast.error('Erro', error.message || 'Falha ao salvar')
    } finally {
      saving.value = false
    }
  }

  return {
    form,
    editing,
    saving,
    startEdit,
    cancelEdit,
    onSave,
    resetForm,
  }
}
