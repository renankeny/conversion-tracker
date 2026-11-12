<template>
  <div class="page-header">
    <div>
      <div class="title-row">
        <div class="title-wrap">
          <h1 class="page-title">Dashboard</h1>
          <span v-if="selectedPixelObj" class="live-pill" title="Rastreamento em tempo real ativo">
            <span class="pulse-dot" />
            <span>LIVE</span>
          </span>
        </div>

        <!-- Link público de visualização (por pixel) -->
        <div v-if="selectedPixelObj" class="share-link-bar">
          <template v-if="!shareToken">
            <BaseButton variant="outline" size="sm" :loading="shareLinkLoading" @click="generateShareLink">
              Gerar link do cliente
            </BaseButton>
          </template>
          <template v-else>
            <BaseButton variant="ghost" size="sm" @click="linkVisible = !linkVisible">
              {{ linkVisible ? 'Ocultar link' : 'Ver link do cliente' }}
            </BaseButton>
            <template v-if="linkVisible">
              <code class="share-link-value">{{ publicDashboardUrl }}</code>
              <BaseButton variant="outline" size="sm" @click="copyShareLink">Copiar</BaseButton>
              <BaseButton variant="ghost" size="sm" :loading="shareLinkLoading" @click="revokeShareLink">Desativar</BaseButton>
            </template>
          </template>
        </div>
      </div>
      <p class="page-subtitle">{{ headerSubtitle }}</p>
    </div>

    <div class="header-controls">
      <PixelDropdown v-model="selectedPixel" :allow-all="false" />
      <BaseDropdown
        v-model="dashboardStore.trafficSource"
        :options="sourceOptions"
      />
      <PeriodSelector
        v-model="dashboardStore.period"
        v-model:customFrom="dashboardStore.customFrom"
        v-model:customTo="dashboardStore.customTo"
      />
      <BaseButton
        variant="ghost"
        size="sm"
        :title="dashboardStore.privacyMode ? 'Mostrar valores financeiros' : 'Ocultar valores financeiros'"
        @click="dashboardStore.togglePrivacyMode()"
      >
        <EyeOff v-if="dashboardStore.privacyMode" :size="15" />
        <Eye v-else :size="15" />
      </BaseButton>
      <BaseButton variant="outline" size="sm" :loading="loading" @click="$emit('refresh')">
        Atualizar
      </BaseButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Eye, EyeOff } from 'lucide-vue-next'
import PeriodSelector from '~/components/dashboard/PeriodSelector.vue'
import { useDashboardStore } from '~/stores/useDashboardStore'
import type { Pixel } from '#shared/types'

const props = defineProps<{
  selectedPixelObj: Pixel | null
  headerSubtitle: string
  loading: boolean
}>()

defineEmits<{ refresh: [] }>()

const dashboardStore = useDashboardStore()
const pixelsApi = usePixels()
const pixelStore = usePixelStore()
const toast = useToast()

const selectedPixel = computed({
  get: () => dashboardStore.selectedPixel,
  set: (v: string) => { dashboardStore.selectedPixel = v },
})

const sourceOptions = [
  { value: 'all', label: 'Todas as fontes' },
  { value: 'meta', label: 'Meta Ads' },
  { value: 'google', label: 'Google Ads' },
]

const shareLinkLoading = ref(false)
const linkVisible = ref(false)
const shareToken = computed(() => props.selectedPixelObj?.dashboard_share_token || '')
const publicDashboardUrl = computed(() => {
  if (!shareToken.value) return ''
  return `${useRuntimeConfig().public.appUrl}/cliente/${shareToken.value}`
})

watch(shareToken, () => { linkVisible.value = false })

const generateShareLink = async () => {
  if (!props.selectedPixelObj) return
  shareLinkLoading.value = true
  try {
    const res = await pixelsApi.generateDashboardShareLink(props.selectedPixelObj.id)
    pixelStore.upsertPixel({ ...props.selectedPixelObj, dashboard_share_token: res.dashboard_share_token })
    toast.success('Sucesso', 'Link público gerado.')
  } catch {
    toast.error('Erro', 'Não foi possível gerar o link público.')
  } finally {
    shareLinkLoading.value = false
  }
}

const copyShareLink = async () => {
  if (!publicDashboardUrl.value) return
  await navigator.clipboard.writeText(publicDashboardUrl.value)
  toast.success('Copiado', 'Link copiado pra área de transferência.')
}

const revokeShareLink = async () => {
  if (!props.selectedPixelObj) return
  shareLinkLoading.value = true
  try {
    await pixelsApi.revokeDashboardShareLink(props.selectedPixelObj.id)
    pixelStore.upsertPixel({ ...props.selectedPixelObj, dashboard_share_token: '' })
    toast.success('Sucesso', 'Link público desativado.')
  } catch {
    toast.error('Erro', 'Não foi possível desativar o link público.')
  } finally {
    shareLinkLoading.value = false
  }
}
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  gap: 1rem;
  flex-wrap: wrap;
}

.title-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.title-wrap {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.page-title {
  font-family: 'Instrument Sans', sans-serif;
  font-size: var(--fs-3xl);
  font-weight: var(--fw-bold);
  color: var(--text-strong);
  letter-spacing: -0.02em;
  margin-bottom: 0;
}

.live-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.15rem 0.45rem;
  background: var(--brand-soft);
  border: 1px solid var(--brand-border);
  border-radius: 9999px;
  font-size: var(--fs-2xs);
  font-weight: var(--fw-bold);
  color: var(--brand);
  letter-spacing: 0.06em;
}

.pulse-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--brand);
  box-shadow: 0 0 6px var(--brand);
  animation: livePulse 1.8s infinite;
}

@keyframes livePulse {
  0% { transform: scale(0.95); opacity: 0.8; }
  50% { transform: scale(1.3); opacity: 1; box-shadow: 0 0 10px var(--brand); }
  100% { transform: scale(0.95); opacity: 0.8; }
}

.page-subtitle {
  color: var(--text-3);
  font-size: var(--fs-base);
  margin-top: 0.25rem;
}

.header-controls {
  display: flex;
  gap: 0.625rem;
  align-items: center;
  flex-wrap: wrap;
}

.share-link-bar {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.share-link-value {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: var(--fs-sm);
  background: var(--surface-soft);
  border: 1px solid var(--border-soft);
  border-radius: 6px;
  padding: 0.25rem 0.5rem;
  color: var(--text-1);
  overflow-wrap: anywhere;
}

@media (max-width: 640px) {
  .page-header { align-items: stretch; }
  .header-controls { justify-content: flex-start; }
  .share-link-value { width: 100%; }
}
</style>
