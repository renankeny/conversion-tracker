<template>
  <section class="account-stack">
    <div class="card">
      <h3 class="card-title">Meu perfil</h3>
      <p class="card-hint">
        Esses dados aparecem no painel (sidebar, listas) e ajudam o time a
        identificar quem está logado. Não afetam autenticação.
      </p>
      <form class="form" @submit.prevent="onSaveProfile">
        <BaseInput
          v-model="profileForm.name"
          type="text"
          label="Nome"
          placeholder="Como deve aparecer no painel"
        />
        <BaseInput
          :model-value="user?.email || ''"
          type="email"
          label="E-mail (login)"
          disabled
          hint="O e-mail é o identificador único de login e não pode ser alterado por aqui."
        />
        <BaseInput
          v-model="profileForm.phone"
          type="tel"
          label="Telefone"
          placeholder="(11) 99999-9999"
        />
        <div class="form-actions">
          <BaseButton variant="primary" type="submit" :loading="savingProfile">
            Salvar perfil
          </BaseButton>
        </div>
      </form>
    </div>

    <div class="card">
      <h3 class="card-title">Inteligência Artificial (OpenAI)</h3>
      <p class="card-hint">
        Configure a sua chave da OpenAI para utilizar recursos de automação com Inteligência Artificial no {{ brandName }} (ex: Auto-Mapping de Checkouts).
      </p>
      <form class="form" @submit.prevent="onSavePrefs">
        <BaseInput
          v-model="prefsForm.openai_api_key"
          type="password"
          label="OpenAI API Key (sk-...)"
          :placeholder="hasOpenAiKey ? '•••••••••••••••• (chave configurada — digite para substituir)' : 'sk-proj-...'"
          hint="A chave fica salva em suas preferências pessoais e não é compartilhada."
        />
        <div class="form-actions">
          <BaseButton variant="primary" type="submit" :loading="savingPrefs">
            Salvar chave
          </BaseButton>
        </div>
      </form>
    </div>

    <div class="card">
      <h3 class="card-title">Trocar minha senha</h3>
      <p class="card-hint">
        Logado como <strong>{{ user?.email }}</strong>. A nova senha precisa ter no mínimo 6 caracteres.
      </p>
      <form class="form" @submit.prevent="onChangePassword">
        <BaseInput
          v-model="passwordForm.current"
          type="password"
          label="Senha atual"
          required
        />
        <BaseInput
          v-model="passwordForm.next"
          type="password"
          label="Nova senha"
          required
        />
        <BaseInput
          v-model="passwordForm.confirm"
          type="password"
          label="Confirmar nova senha"
          :error="passwordError"
          required
        />
        <div class="form-actions">
          <BaseButton variant="primary" type="submit" :loading="changingPassword">
            Salvar nova senha
          </BaseButton>
        </div>
      </form>
    </div>
  </section>
</template>

<script setup lang="ts">
import { errorMessage } from '~/utils/errorMessage'

const settings = useSettings()
const { user, hasOpenAiKey, setUser } = useAuth()
const { brandName } = useBranding()
const toast = useToast()

const profileForm = reactive({
  name: user.value?.name || '',
  phone: user.value?.phone || '',
})
const savingProfile = ref(false)

watch(
  () => user.value,
  (u) => {
    if (!u) return
    profileForm.name = u.name || ''
    profileForm.phone = u.phone || ''
  },
  { immediate: true }
)

const onSaveProfile = async () => {
  savingProfile.value = true
  try {
    const updated = await settings.updateMyProfile({
      name: profileForm.name.trim(),
      phone: profileForm.phone.trim(),
    })
    if (user.value) {
      setUser({ ...user.value, name: updated.name, phone: updated.phone })
    }
    toast.success('Sucesso', 'Perfil atualizado')
  } catch (error: unknown) {
    toast.error('Erro', errorMessage(error, 'Falha ao salvar perfil'))
  } finally {
    savingProfile.value = false
  }
}

const prefsForm = reactive({
  openai_api_key: String(user.value?.preferences?.openai_api_key || ''),
})
const savingPrefs = ref(false)

watch(
  () => user.value?.preferences,
  (p) => {
    if (p) prefsForm.openai_api_key = String(p.openai_api_key || '')
  },
  { immediate: true }
)

const onSavePrefs = async () => {
  savingPrefs.value = true
  try {
    const prefs = { ...(user.value?.preferences || {}), openai_api_key: prefsForm.openai_api_key.trim() }
    const res = await settings.updateMyPreferences(prefs)
    if (user.value) {
      setUser({ ...user.value, preferences: res.preferences })
    }
    prefsForm.openai_api_key = ''
    toast.success('Sucesso', 'Preferências salvas')
  } catch (error: unknown) {
    toast.error('Erro', errorMessage(error, 'Falha ao salvar preferências'))
  } finally {
    savingPrefs.value = false
  }
}

const passwordForm = reactive({ current: '', next: '', confirm: '' })
const passwordError = ref('')
const changingPassword = ref(false)

const onChangePassword = async () => {
  passwordError.value = ''
  if (passwordForm.next !== passwordForm.confirm) {
    passwordError.value = 'As senhas não conferem'
    return
  }
  if (passwordForm.next.length < 6) {
    passwordError.value = 'Mínimo 6 caracteres'
    return
  }
  changingPassword.value = true
  try {
    await settings.changeMyPassword({
      current_password: passwordForm.current,
      new_password: passwordForm.next,
    })
    toast.success('Senha atualizada', 'Use a nova no próximo login')
    passwordForm.current = ''
    passwordForm.next = ''
    passwordForm.confirm = ''
  } catch (error: unknown) {
    toast.error('Erro', errorMessage(error, 'Não foi possível alterar a senha'))
  } finally {
    changingPassword.value = false
  }
}
</script>

<style scoped>
.account-stack { display: flex; flex-direction: column; gap: 1rem; }
.card { background: var(--surface-2); border: 1px solid var(--border); border-radius: 12px; padding: 1.5rem; box-shadow: var(--shadow-card); }
.card-title { font-family: var(--font-sans); font-size: var(--fs-xl); font-weight: var(--fw-semibold); color: var(--text-strong); margin-bottom: 0.5rem; }

.card-hint {
  font-size: var(--fs-md);
  color: var(--text-2);
  margin-bottom: 1rem;
  line-height: 1.5;
}

.form {
  display: flex;
  flex-direction: column;
  gap: 0.875rem;
  max-width: 420px;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 0.5rem;
}
</style>

