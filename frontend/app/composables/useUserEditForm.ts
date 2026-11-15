import { ref, reactive, watch, type Ref, type ComputedRef } from 'vue'
import type { TrackedUser, TrackedUserUpdateInput } from '#shared/types'
import { decodeUtm } from '~/components/users/userFormatters'
import { errorMessage } from '~/utils/errorMessage'

const EDITABLE_FIELDS: (keyof TrackedUserUpdateInput)[] = [
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

const createEmptySnapshot = (): Required<TrackedUserUpdateInput> => ({
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

export const useUserEditForm = (
  userRef: Ref<TrackedUser | null> | ComputedRef<TrackedUser | null>,
  onSavedCallback?: (updated: TrackedUser) => void
) => {
  const trackedUsers = useTrackedUsers()
  const toast = useToast()

  const editing = ref(false)
  const saving = ref(false)

  const form = reactive<Required<TrackedUserUpdateInput>>(createEmptySnapshot())
  let initialSnapshot: Record<keyof TrackedUserUpdateInput, string> = createEmptySnapshot()

  const resetForm = (u: TrackedUser | null) => {
    const nextSnapshot = createEmptySnapshot()
    if (u) {
      for (const f of EDITABLE_FIELDS) {
        const raw = u[f as keyof TrackedUser]
        let val = typeof raw === 'string' ? raw : ''
        if (val && f.includes('utm_')) {
          val = decodeUtm(val)
        }
        nextSnapshot[f] = val
      }
    }
    initialSnapshot = { ...nextSnapshot }
    Object.assign(form, nextSnapshot)
  }

  watch(
    () => userRef.value,
    (u, prev) => {
      if (!u || u.id !== prev?.id) {
        editing.value = false
        resetForm(u)
      } else if (!editing.value) {
        resetForm(u)
      }
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

    const patch: TrackedUserUpdateInput = {}
    for (const k of EDITABLE_FIELDS) {
      if (form[k] !== initialSnapshot[k]) {
        patch[k] = form[k]
      }
    }

    if (Object.keys(patch).length === 0) {
      editing.value = false
      return user
    }

    saving.value = true
    try {
      const updated = await trackedUsers.update(user.id, patch)
      toast.success('Sucesso', 'Usuário atualizado com sucesso')
      editing.value = false
      resetForm(updated)
      if (onSavedCallback) onSavedCallback(updated)
      return updated
    } catch (error: unknown) {
      toast.error('Erro', errorMessage(error, 'Falha ao salvar'))
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
