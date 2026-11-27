<template>
  <div class="step-done">
    <div class="done-hero">
      <div class="done-icon">
        <CheckCircle2 :size="24" />
      </div>
      <div>
        <h3 class="done-title">{{ createdAccount?.name || clientName }} pronto para rodar!</h3>
        <p class="done-sub">
          Todas as integrações selecionadas foram configuradas e a empresa já está ativa no seu painel.
        </p>
      </div>
    </div>

    <div class="summary-strip">
      <span v-if="createdAccount" class="summary-pill ok">Empresa: {{ createdAccount.name }}</span>
      <span v-if="metaAccount?.connected" class="summary-pill ok">Meta Ads conectado</span>
      <span v-else-if="includesMeta" class="summary-pill">Meta Pixel manual</span>
      <span v-if="googleAccount?.connected || googleAccount?.oauth_connected" class="summary-pill ok">Google Ads conectado</span>
      <span v-if="createdPixel" class="summary-pill ok">Pixel: {{ createdPixel.label || createdPixel.name }}</span>
      <span v-if="createdInstance" class="summary-pill" :class="{ ok: createdInstance.connection_status === 'OPEN' }">
        WhatsApp: {{ createdInstance.connection_status === 'OPEN' ? 'Conectado' : 'Aguardando conexão' }}
      </span>
    </div>

    <div v-if="includesSite && snippet" class="script-card">
      <div class="card-head">
        <div>
          <h4 class="card-title">Copie e cole este script no site</h4>
          <p class="card-hint">Cole antes da tag <code>&lt;/head&gt;</code> no WordPress, Shopify, Elementor, GTM ou página própria.</p>
        </div>
        <BaseButton variant="primary" size="sm" @click="copySnippet">
          <Check v-if="scriptCopied" :size="14" />
          <Copy v-else :size="14" />
          <span>{{ scriptCopied ? 'Copiado!' : 'Copiar Script' }}</span>
        </BaseButton>
      </div>
      <pre class="code-block"><code>{{ snippet }}</code></pre>
      <div class="checkout-tip">
        <span>Vende por plataforma de checkout (Hotmart, Kiwify, Cakto, Yampi...)?</span>
        <button type="button" class="inline-link" @click="goTo('/integracoes/plataformas')">
          Configurar Webhook de Checkout →
        </button>
      </div>
    </div>

    <div class="quick-actions">
      <div v-if="whatsappShareUrl && createdInstance?.connection_status !== 'OPEN'" class="action-row">
        <div class="action-info">
          <span class="action-title">Link para o cliente conectar o WhatsApp</span>
          <span class="action-desc">Envie para o cliente escanear o QR Code direto do celular dele.</span>
        </div>
        <BaseButton variant="outline" size="sm" @click="copyText(whatsappShareUrl, 'wa')">
          <Check v-if="copiedTag === 'wa'" :size="14" />
          <Link2 v-else :size="14" />
          <span>{{ copiedTag === 'wa' ? 'Copiado!' : 'Copiar link' }}</span>
        </BaseButton>
      </div>

      <div v-if="createdPixel" class="action-row">
        <div class="action-info">
          <span class="action-title">Portal do Cliente (Dashboard LIVE)</span>
          <span class="action-desc">
            {{ portalUrl ? 'Link gerado e pronto para enviar ao cliente.' : 'Gere um link Whitelabel em tempo real para compartilhar resultados com o cliente.' }}
          </span>
        </div>
        <BaseButton v-if="!portalUrl" variant="outline" size="sm" :loading="generatingPortal" @click="generatePortalLink">
          <ExternalLink :size="14" />
          <span>Gerar link</span>
        </BaseButton>
        <BaseButton v-else variant="outline" size="sm" @click="copyText(portalUrl, 'portal')">
          <Check v-if="copiedTag === 'portal'" :size="14" />
          <Copy v-else :size="14" />
          <span>{{ copiedTag === 'portal' ? 'Copiado!' : 'Copiar Portal' }}</span>
        </BaseButton>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { Check, CheckCircle2, Copy, ExternalLink, Link2 } from 'lucide-vue-next'
import { buildPixelSnippet } from '~/utils/apiOrigin'
import { useClientOnboardingWizard } from '~/composables/useClientOnboardingWizard'
import { useDashboardStore } from '~/stores/useDashboardStore'
import { errorMessage } from '~/utils/errorMessage'

const emit = defineEmits<{ finish: [] }>()

const router = useRouter()
const config = useRuntimeConfig()
const { brandName } = useBranding()
const pixelsApi = usePixels()
const dashboardStore = useDashboardStore()
const toast = useToast()

const {
  clientName,
  createdAccount,
  metaAccount,
  googleAccount,
  createdPixel,
  createdInstance,
  includesMeta,
  includesSite,
  closeWizard,
} = useClientOnboardingWizard()

const scriptCopied = ref(false)
const copiedTag = ref('')
const generatingPortal = ref(false)
const portalToken = ref(createdPixel.value?.dashboard_share_token || '')

const snippet = computed(() =>
  createdPixel.value
    ? buildPixelSnippet(
        config.public.apiUrl as string,
        createdPixel.value.name,
        createdPixel.value.public_token || undefined,
        brandName.value,
      )
    : '',
)

const whatsappShareUrl = computed(() => {
  if (!createdInstance.value?.share_token || typeof window === 'undefined') return ''
  return `${window.location.origin}/conectar-whatsapp/${createdInstance.value.share_token}`
})

const portalUrl = computed(() => {
  if (!portalToken.value || typeof window === 'undefined') return ''
  return `${window.location.origin}/cliente/${portalToken.value}`
})

const copySnippet = async () => {
  if (!snippet.value) return
  await navigator.clipboard.writeText(snippet.value)
  scriptCopied.value = true
  setTimeout(() => { scriptCopied.value = false }, 2200)
  toast.success('Script copiado!', 'Cole antes de </head> no site do cliente.')
}

const copyText = async (url: string, tag: string) => {
  await navigator.clipboard.writeText(url)
  copiedTag.value = tag
  setTimeout(() => { copiedTag.value = '' }, 2200)
  toast.success('Link copiado!', 'Pronto para enviar ao cliente.')
}

const generatePortalLink = async () => {
  if (!createdPixel.value) return
  generatingPortal.value = true
  try {
    const res = await pixelsApi.generateDashboardShareLink(createdPixel.value.id)
    portalToken.value = res.dashboard_share_token
    if (portalUrl.value) {
      await navigator.clipboard.writeText(portalUrl.value).catch(() => {})
      copiedTag.value = 'portal'
      setTimeout(() => { copiedTag.value = '' }, 2200)
    }
    toast.success('Portal do Cliente gerado!', 'Link copiado para a área de transferência.')
  } catch (err: unknown) {
    toast.error('Erro ao gerar link', errorMessage(err, 'Tente novamente na tela de Pixels.'))
  } finally {
    generatingPortal.value = false
  }
}

const goTo = async (path: string) => {
  if (createdPixel.value) dashboardStore.setSelectedPixel(createdPixel.value.name)
  closeWizard()
  await router.push(path)
}

const submit = async () => {
  if (createdPixel.value) dashboardStore.setSelectedPixel(createdPixel.value.name)
  emit('finish')
  await router.push('/')
}

defineExpose({ submit })
</script>

<style scoped>
.step-done { display: flex; flex-direction: column; gap: 1.15rem; }
.done-hero { display: flex; align-items: flex-start; gap: 0.85rem; }
.done-icon {
  width: 42px; height: 42px; border-radius: 11px; flex-shrink: 0;
  background: var(--success-soft); color: var(--success); border: 1px solid var(--success);
  display: flex; align-items: center; justify-content: center;
}
.done-title { font-family: var(--font-heading); font-size: var(--fs-xl); font-weight: var(--fw-bold); color: var(--text-strong); margin: 0; }
.done-sub { font-size: var(--fs-sm); color: var(--text-2); margin: 0.2rem 0 0; line-height: 1.45; }
.summary-strip { display: flex; flex-wrap: wrap; gap: 0.45rem; }
.summary-pill {
  padding: 0.28rem 0.65rem; border-radius: 999px; font-size: var(--fs-xs); font-weight: var(--fw-medium);
  background: var(--surface-soft); border: 1px solid var(--border-soft); color: var(--text-2);
}
.summary-pill.ok { background: var(--success-soft); border-color: var(--success); color: var(--success); font-weight: var(--fw-semibold); }
.script-card {
  display: flex; flex-direction: column; gap: 0.7rem; padding: 1rem;
  border-radius: 11px; background: var(--surface-deep); border: 1px solid var(--border);
}
.card-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 0.75rem; flex-wrap: wrap; }
.card-title { font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--text-strong); margin: 0; }
.card-hint { font-size: var(--fs-xs); color: var(--text-3); margin: 0.15rem 0 0; }
.card-hint code { font-family: var(--font-mono); color: var(--text-2); }
.code-block {
  margin: 0; padding: 0.85rem; border-radius: 8px; background: var(--surface-2); border: 1px solid var(--border-soft);
  font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--text-1); overflow-x: auto; white-space: pre; line-height: 1.55;
}
.checkout-tip { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; flex-wrap: wrap; font-size: var(--fs-xs); color: var(--text-3); }
.inline-link { background: transparent; border: none; padding: 0; color: var(--brand); font-size: var(--fs-xs); font-weight: var(--fw-semibold); cursor: pointer; }
.inline-link:hover { text-decoration: underline; }
.quick-actions { display: flex; flex-direction: column; gap: 0.6rem; }
.action-row {
  display: flex; align-items: center; justify-content: space-between; gap: 0.75rem; flex-wrap: wrap;
  padding: 0.8rem 0.95rem; border-radius: 10px; background: var(--surface-soft); border: 1px solid var(--border-soft);
}
.action-info { display: flex; flex-direction: column; gap: 0.15rem; min-width: 0; flex: 1; }
.action-title { font-size: var(--fs-sm); font-weight: var(--fw-semibold); color: var(--text-strong); }
.action-desc { font-size: var(--fs-xs); color: var(--text-3); }
</style>
