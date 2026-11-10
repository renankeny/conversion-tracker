<template>
  <section class="card">
    <h3 class="card-title">Base GeoIP</h3>
    <p class="card-hint">
      A base GeoLite2-City da MaxMind enriquece os <code>tracked_users</code> com
      país, estado, cidade e CEP. Sem ela o sistema funciona, só não tem geo.
    </p>

    <div v-if="loadingGeoIP" class="muted">Carregando...</div>

    <div v-else class="geoip-status">
      <div class="status-grid">
        <div class="status-item">
          <span class="status-label">Status</span>
          <BaseBadge :variant="geoipStatusVariant">
            {{ geoipStatusLabel }}
          </BaseBadge>
        </div>
        <div class="status-item">
          <span class="status-label">License Key</span>
          <BaseBadge :variant="geoip?.license_key_set ? 'success' : 'warning'">
            {{ geoip?.license_key_set ? 'Configurada' : 'Não configurada' }}
          </BaseBadge>
        </div>
        <div class="status-item">
          <span class="status-label">Path</span>
          <code>{{ geoip?.path || '—' }}</code>
        </div>
        <div v-if="geoip?.size_bytes" class="status-item">
          <span class="status-label">Tamanho</span>
          <span>{{ formatBytes(geoip.size_bytes) }}</span>
        </div>
        <div v-if="geoip?.modified_at" class="status-item">
          <span class="status-label">Atualizado em</span>
          <span>{{ formatDate(geoip.modified_at) }}</span>
        </div>
      </div>

      <div class="geoip-actions">
        <BaseButton
          variant="primary"
          :loading="refreshingGeoIP"
          :disabled="!geoip?.license_key_set"
          @click="onRefreshGeoIP"
        >
          Baixar / atualizar base agora
        </BaseButton>
        <BaseButton variant="outline" size="sm" @click="loadGeoIP">
          Recarregar status
        </BaseButton>
      </div>

      <p v-if="!geoip?.license_key_set" class="hint-warning">
        Configure <code>MAXMIND_LICENSE_KEY</code> nas envs do servidor.
        A chave gratuita é obtida em
        <a href="https://www.maxmind.com/en/geolite2/signup" target="_blank" rel="noopener">
          maxmind.com/en/geolite2/signup
        </a>.
      </p>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { GeoIPStatus } from '#shared/types'

const settings = useSettings()
const toast = useToast()

const geoip = ref<GeoIPStatus | null>(null)
const loadingGeoIP = ref(false)
const refreshingGeoIP = ref(false)

const geoipStatusVariant = computed(() => {
  if (!geoip.value) return 'default'
  if (geoip.value.loaded) return 'success'
  if (geoip.value.exists) return 'warning'
  return 'danger'
})

const geoipStatusLabel = computed(() => {
  if (!geoip.value) return ''
  if (geoip.value.loaded) return 'Carregada'
  if (geoip.value.exists) return 'Arquivo existe mas não carregou'
  return 'Não instalada'
})

const formatDate = (iso: string): string =>
  iso ? new Date(iso).toLocaleString('pt-BR', { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit' }) : ''

const formatBytes = (bytes: number): string => {
  if (!bytes) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  let i = 0
  let v = bytes
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${v.toFixed(1)} ${units[i]}`
}

const loadGeoIP = async () => {
  loadingGeoIP.value = true
  try {
    geoip.value = await settings.geoipStatus()
  } catch (error: any) {
    toast.error('Erro', error.message || 'Falha ao consultar GeoIP')
  } finally {
    loadingGeoIP.value = false
  }
}

const onRefreshGeoIP = async () => {
  refreshingGeoIP.value = true
  try {
    const res = await settings.geoipRefresh()
    if (res.installed) {
      toast.success(
        'GeoIP atualizada',
        res.reloaded ? 'Base recarregada com sucesso' : 'Base salva, mas não recarregada'
      )
    }
    await loadGeoIP()
  } catch (error: any) {
    toast.error('Erro', error.message || 'Falha ao baixar GeoIP')
  } finally {
    refreshingGeoIP.value = false
  }
}

onMounted(loadGeoIP)
</script>

<style scoped>
.card { background: var(--surface-2); border: 1px solid var(--border); border-radius: 12px; padding: 1.5rem; box-shadow: var(--shadow-card); }
.card-title { font-family: 'Instrument Sans', sans-serif; font-size: var(--fs-xl); font-weight: var(--fw-semibold); color: var(--text-strong); margin-bottom: 0.5rem; }
.card-hint { font-size: var(--fs-md); color: var(--text-2); margin-bottom: 1rem; line-height: 1.5; }
.card-hint code { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; background: var(--surface-soft); padding: 0.0625rem 0.375rem; border-radius: 4px; }
.muted { color: var(--text-3); font-size: var(--fs-md); font-style: italic; }
.geoip-status { display: flex; flex-direction: column; gap: 1rem; }
.status-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 0.75rem 1rem; }
.status-item { display: flex; flex-direction: column; gap: 0.25rem; }
.status-label { font-size: var(--fs-xs); text-transform: uppercase; letter-spacing: 0.05em; color: var(--text-3); font-weight: var(--fw-semibold); }
.status-item code { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: var(--fs-sm); background: var(--surface-soft); padding: 0.125rem 0.5rem; border-radius: 4px; word-break: break-all; }
.geoip-actions { display: flex; gap: 0.5rem; flex-wrap: wrap; margin-top: 0.5rem; }
.hint-warning { font-size: var(--fs-base); color: var(--warning); background: var(--warning-soft); border: 1px solid var(--warning); border-radius: 8px; padding: 0.625rem 0.875rem; }
.hint-warning code { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; background: var(--surface-soft); padding: 0 0.25rem; border-radius: 3px; }
.hint-warning a { color: var(--warning); text-decoration: underline; }
</style>

