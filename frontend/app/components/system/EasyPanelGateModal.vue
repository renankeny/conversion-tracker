<template>
  <ClientOnly>
    <Teleport to="body">
      <Transition name="ep-fade">
        <div v-if="shouldShowGate" class="ep-gate-overlay">
          <div class="ep-gate-card">
            <div class="ep-gate-header">
              <div class="ep-gate-icon">
                <Server :size="20" />
              </div>
              <div>
                <span class="ep-gate-badge">Atualizações Automáticas</span>
                <h2 class="ep-gate-title">Conecte seu EasyPanel</h2>
                <p class="ep-gate-subtitle">
                  Vincule a URL e a API Key do seu EasyPanel para receber melhorias e correções da plataforma em 1 clique.
                </p>
              </div>
            </div>

            <div v-if="!noEasyPanel" class="ep-gate-steps">
              <div class="ep-step-row">
                <span class="ep-step-num">1</span>
                <span>Abra seu <strong>EasyPanel</strong> e clique em <strong>Settings</strong> (canto inferior esquerdo).</span>
              </div>
              <div class="ep-step-row">
                <span class="ep-step-num">2</span>
                <span>Acesse a aba <strong>Users</strong>, gere ou copie seu <strong>API Token</strong>.</span>
              </div>
              <div class="ep-step-row">
                <span class="ep-step-num">3</span>
                <span>Cole os dados abaixo. Seus eventos, pixels e webhooks continuam rodando normalmente.</span>
              </div>
            </div>

            <form class="ep-gate-form" @submit.prevent="onSubmit">
              <template v-if="!noEasyPanel">
                <BaseInput
                  v-model="panelUrl"
                  label="URL do seu EasyPanel"
                  placeholder="Ex: https://easypanel.seudominio.com.br"
                  hint="Endereço onde você acessa o painel do EasyPanel da sua VPS."
                  required
                />

                <BaseInput
                  v-model="apiToken"
                  type="password"
                  label="API Key (Token do EasyPanel)"
                  placeholder="Cole aqui o API Token gerado no EasyPanel"
                  required
                />
              </template>

              <div class="ep-skip-box">
                <BaseCheckbox
                  v-model="noEasyPanel"
                  size="sm"
                  label="Não uso EasyPanel"
                  hint="Marque se a sua VPS utiliza Docker puro, Portainer ou outro gerenciador (não será solicitado novamente)."
                />
              </div>

              <div v-if="errorMsg" class="ep-gate-error">
                {{ errorMsg }}
              </div>

              <div class="ep-gate-actions">
                <BaseButton type="button" variant="ghost" size="sm" :disabled="submitting" @click="onLogout">
                  Sair da conta
                </BaseButton>
                <BaseButton type="submit" variant="primary" size="lg" :loading="submitting">
                  {{ noEasyPanel ? 'Confirmar & Liberar Painel' : 'Validar & Liberar Painel' }}
                </BaseButton>
              </div>
            </form>
          </div>
        </div>
      </Transition>
    </Teleport>
  </ClientOnly>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Server } from 'lucide-vue-next'
import { useEasyPanelGate } from '~/composables/useEasyPanelGate'
import { useToast } from '~/composables/useToast'
import { useAuthStore } from '~/stores/useAuthStore'

const route = useRoute()
const authStore = useAuthStore()
const { add: addToast } = useToast()
const {
  checked,
  linked,
  hintUrl,
  submitting,
  errorMsg,
  checkEasyPanelStatus,
  connectEasyPanel,
  resetGate,
} = useEasyPanelGate()

const panelUrl = ref('')
const apiToken = ref('')
const noEasyPanel = ref(false)

const isBypassedRoute = computed(() => {
  const p = route.path || ''
  return p.startsWith('/login') || p.startsWith('/setup') || p.startsWith('/public') || p.startsWith('/conectar-whatsapp')
})

const shouldShowGate = computed(() => {
  return authStore.isAuthenticated && !isBypassedRoute.value && checked.value && !linked.value
})

watch(noEasyPanel, () => {
  errorMsg.value = ''
})

watch(
  () => authStore.isAuthenticated,
  async (authed) => {
    if (authed) {
      await checkEasyPanelStatus()
      if (!panelUrl.value && hintUrl.value) {
        panelUrl.value = hintUrl.value
      }
    } else {
      resetGate()
    }
  },
  { immediate: true },
)

watch(hintUrl, (val) => {
  if (val && !panelUrl.value) {
    panelUrl.value = val
  }
})

const onSubmit = async () => {
  const ok = await connectEasyPanel(panelUrl.value, apiToken.value, noEasyPanel.value)
  if (ok) {
    apiToken.value = ''
    addToast({
      type: 'success',
      title: noEasyPanel.value ? 'Preferência salva!' : 'EasyPanel conectado!',
      message: noEasyPanel.value
        ? 'Acesso liberado sem EasyPanel.'
        : 'Sua instância agora recebe atualizações automáticas.',
    })
  }
}

const onLogout = async () => {
  resetGate()
  await authStore.logout()
}
</script>

<style scoped>
.ep-gate-overlay {
  position: fixed; inset: 0; z-index: 9500; padding: 1.25rem; overflow-y: auto;
  background: var(--overlay-bg); backdrop-filter: blur(8px);
  display: flex; align-items: center; justify-content: center;
}
.ep-gate-card {
  width: 100%; max-width: 520px; padding: 1.5rem; border-radius: 16px;
  background: var(--surface-2); border: 1px solid var(--border); box-shadow: var(--shadow-elevated);
  display: flex; flex-direction: column; gap: 1.15rem;
}
.ep-gate-header { display: flex; align-items: flex-start; gap: 0.9rem; }
.ep-gate-icon {
  width: 42px; height: 42px; border-radius: 11px; flex-shrink: 0;
  background: var(--brand-soft); border: 1px solid var(--brand-border); color: var(--brand);
  display: flex; align-items: center; justify-content: center;
}
.ep-gate-badge {
  display: inline-block; font-size: var(--fs-2xs); font-weight: var(--fw-semibold);
  color: var(--brand); text-transform: uppercase; letter-spacing: 0.05em;
}
.ep-gate-title {
  font-family: var(--font-heading); font-size: var(--fs-xl); font-weight: var(--fw-bold);
  color: var(--text-strong); margin-top: 0.1rem;
}
.ep-gate-subtitle { font-size: var(--fs-sm); color: var(--text-2); line-height: 1.5; margin-top: 0.25rem; }
.ep-gate-steps {
  display: flex; flex-direction: column; gap: 0.5rem; padding: 0.85rem 1rem;
  border-radius: 10px; background: var(--surface-1); border: 1px solid var(--border-soft);
}
.ep-step-row { display: flex; align-items: center; gap: 0.65rem; font-size: var(--fs-xs); color: var(--text-2); }
.ep-step-row strong { color: var(--text-strong); font-weight: var(--fw-semibold); }
.ep-step-num {
  width: 20px; height: 20px; border-radius: 999px; flex-shrink: 0;
  background: var(--brand-soft); border: 1px solid var(--brand-border); color: var(--brand);
  font-size: var(--fs-2xs); font-weight: var(--fw-bold);
  display: inline-flex; align-items: center; justify-content: center;
}
.ep-gate-form { display: flex; flex-direction: column; gap: 0.9rem; }
.ep-skip-box {
  padding: 0.75rem 0.9rem; border-radius: 10px;
  background: var(--surface-1); border: 1px solid var(--border-soft);
}
.ep-gate-error {
  padding: 0.65rem 0.85rem; border-radius: 8px; font-size: var(--fs-xs);
  background: var(--danger-soft); border: 1px solid var(--danger); color: var(--danger);
}
.ep-gate-actions {
  display: flex; align-items: center; justify-content: space-between; gap: 0.75rem;
  padding-top: 0.5rem; border-top: 1px solid var(--border-soft);
}
.ep-fade-enter-active, .ep-fade-leave-active { transition: opacity 0.2s ease; }
.ep-fade-enter-from, .ep-fade-leave-to { opacity: 0; }
</style>
