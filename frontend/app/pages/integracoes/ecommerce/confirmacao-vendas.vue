<template>
  <div class="sales-confirmation-page">
    <div class="page-header">
      <div>
        <h1 class="page-title">Confirmação de Vendas</h1>
        <p class="page-subtitle">
          Gerencie Pix Gerados e force o envio de compras (Purchase) para a Meta.
        </p>
      </div>
      <BaseButton variant="outline" @click="fetchSales">Atualizar</BaseButton>
    </div>

    <div v-if="loading && !sales.length" class="loading-state">
      <div class="spinner" />
      <p>Carregando vendas...</p>
    </div>

    <div v-else-if="!sales.length" class="empty-state">
      <h2 class="empty-title">Nenhuma venda encontrada</h2>
      <p class="empty-description">Não há Pix gerados ou compras registradas recentemente.</p>
    </div>

    <div v-else class="sales-table-container overflow-x-auto">
      <table class="data-table">
        <thead>
          <tr>
            <th>Data / Hora</th>
            <th>Pixel</th>
            <th>Cliente</th>
            <th>Valor</th>
            <th>Status</th>
            <th>Ações</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="sale in sales" :key="sale.event_id">
            <td>{{ formatDateTimeSP(sale.created_at) }}</td>
            <td><span class="pixel-name">{{ sale.pixel_name }}</span></td>
            <td>
              <div class="text-strong">{{ sale.first_name }} {{ sale.last_name }}</div>
              <div class="text-weak">{{ sale.email }}</div>
              <div class="text-weak">{{ sale.phone }}</div>
            </td>
            <td>{{ formatCurrency(sale.purchase_value) }}</td>
            <td>
              <span v-if="sale.event_name === 'Purchase'" class="badge badge-success">
                <svg class="badge-icon" viewBox="0 0 16 16" fill="currentColor">
                  <path fill-rule="evenodd" d="M12.416 3.376a.75.75 0 0 1 .208 1.04l-5 7.5a.75.75 0 0 1-1.154.114l-3-3a.75.75 0 0 1 1.06-1.06l2.353 2.353 4.493-6.74a.75.75 0 0 1 1.04-.207Z" clip-rule="evenodd" />
                </svg>
                Compra Confirmada
              </span>
              <span v-else class="badge badge-warning">
                <svg class="badge-icon" viewBox="0 0 16 16" fill="currentColor">
                  <path fill-rule="evenodd" d="M1 8a7 7 0 1 1 14 0A7 7 0 0 1 1 8Zm7.75-4.25a.75.75 0 0 0-1.5 0V8c0 .414.336.75.75.75h3.25a.75.75 0 0 0 0-1.5h-2.5V3.75Z" clip-rule="evenodd" />
                </svg>
                Pix Pendente
              </span>
            </td>
            <td>
              <div class="actions-cell">
                <BaseButton
                  v-if="sale.event_name !== 'Purchase'"
                  variant="primary"
                  size="sm"
                  :loading="confirming === sale.event_id"
                  @click="confirmSale(sale)"
                >
                  Confirmar Pagamento
                </BaseButton>
                <div v-else class="confirmed-action-group">
                  <span class="confirmed-pill">
                    <svg class="badge-icon" viewBox="0 0 16 16" fill="currentColor">
                      <path fill-rule="evenodd" d="M12.416 3.376a.75.75 0 0 1 .208 1.04l-5 7.5a.75.75 0 0 1-1.154.114l-3-3a.75.75 0 0 1 1.06-1.06l2.353 2.353 4.493-6.74a.75.75 0 0 1 1.04-.207Z" clip-rule="evenodd" />
                    </svg>
                    Enviado
                  </span>
                  <BaseButton
                    variant="ghost"
                    size="sm"
                    :loading="confirming === sale.event_id"
                    title="Reenviar evento Purchase para a Meta"
                    @click="confirmSale(sale)"
                  >
                    Reenviar
                  </BaseButton>
                </div>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <ConfirmModal
      v-model="confirmModalOpen"
      title="Confirmar Venda"
      :message="saleToConfirm ? `Deseja enviar um evento de Compra (Purchase) no valor de ${formatCurrency(saleToConfirm.purchase_value)} para o Facebook?` : ''"
      confirm-label="Confirmar e Enviar"
      variant="primary"
      :loading="confirming === (saleToConfirm?.event_id || '')"
      @confirm="executeConfirmSale"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import ConfirmModal from '~/components/ConfirmModal.vue'
import { useTenantStore } from '~/stores/useTenantStore'
import { formatDateTimeSP } from '~/utils/formatDate'
import { errorMessage } from '~/utils/errorMessage'

interface SaleConfirmationItem {
  event_id: string
  created_at: string
  pixel_name: string
  first_name?: string
  last_name?: string
  email?: string
  phone?: string
  purchase_value: number
  event_name: string
}

definePageMeta({ layout: 'app', middleware: 'auth' })
const { formatTitle } = useBranding()
useHead({ title: computed(() => formatTitle('Confirmação de Vendas')) })

const api = useAPI()
const toast = useToast()
const tenantStore = useTenantStore()

const sales = ref<SaleConfirmationItem[]>([])
const loading = ref(true)
const confirming = ref<string | null>(null)
const confirmModalOpen = ref(false)
const saleToConfirm = ref<SaleConfirmationItem | null>(null)

const fetchSales = async () => {
  loading.value = true
  try {
    const res = await api.get<{ items?: SaleConfirmationItem[] }>('/admin/sales-confirmations?limit=50')
    sales.value = res.items || []
  } catch (err: unknown) {
    toast.error('Erro ao buscar vendas', errorMessage(err, 'Falha ao buscar vendas'))
  } finally {
    loading.value = false
  }
}

const confirmSale = (sale: SaleConfirmationItem) => {
  saleToConfirm.value = sale
  confirmModalOpen.value = true
}

const executeConfirmSale = async () => {
  if (!saleToConfirm.value) return
  const sale = saleToConfirm.value
  confirming.value = sale.event_id
  try {
    await api.post('/admin/sales-confirmations/confirm', {
      event_id: sale.event_id,
      pixel_name: sale.pixel_name,
    })
    const target = sales.value.find((s) => s.event_id === sale.event_id)
    if (target) target.event_name = 'Purchase'
    toast.success('Sucesso', 'Venda confirmada com sucesso! Evento Purchase enviado para a Meta.')
    confirmModalOpen.value = false
    await fetchSales()
  } catch (err: unknown) {
    toast.error('Erro ao confirmar', errorMessage(err, 'Falha ao confirmar venda'))
  } finally {
    confirming.value = null
  }
}

const formatCurrency = (val: number) =>
  new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(val)

watch(() => tenantStore.switchVersion, () => {
  confirmModalOpen.value = false
  saleToConfirm.value = null
  fetchSales()
})

onMounted(() => fetchSales())
</script>

<style scoped>
.sales-confirmation-page { display: flex; flex-direction: column; gap: 1.25rem; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; gap: 1rem; flex-wrap: wrap; }
.page-title { font-family: var(--font-sans); font-size: var(--fs-3xl); font-weight: var(--fw-bold); color: var(--text-strong); margin-bottom: 0.25rem; }
.page-subtitle { font-size: var(--fs-md); color: var(--text-weak); line-height: 1.4; margin: 0; }
.sales-table-container { background: var(--surface-2); border: 1px solid var(--border); border-radius: 12px; box-shadow: var(--shadow-card); overflow: hidden; overflow-x: auto; }
.data-table { width: 100%; border-collapse: collapse; min-width: 1080px; }
.data-table th { background: var(--surface-soft-2); color: var(--text-3); font-size: var(--fs-xs); font-weight: var(--fw-semibold); text-transform: uppercase; letter-spacing: 0.05em; text-align: left; padding: 0.625rem 1rem; border-bottom: 1px solid var(--border); }
.data-table td { padding: 0.75rem 1rem; font-size: var(--fs-md); color: var(--text-1); border-bottom: 1px solid var(--border-soft); vertical-align: middle; }
.data-table tr:hover td { background: var(--surface-soft); }
.badge { display: inline-flex; align-items: center; gap: 0.375rem; padding: 0.25rem 0.625rem; border-radius: 9999px; font-size: var(--fs-sm); font-weight: var(--fw-semibold); white-space: nowrap; }
.badge-icon { width: 0.875rem; height: 0.875rem; flex-shrink: 0; }
.badge-success { background: var(--success-soft); color: var(--success); border: 1px solid var(--success); }
.badge-warning { background: var(--warning-soft); color: var(--warning); border: 1px solid var(--warning); }
.actions-cell, .confirmed-action-group { display: inline-flex; align-items: center; gap: 0.5rem; }
.confirmed-pill { display: inline-flex; align-items: center; gap: 0.25rem; font-size: var(--fs-sm); font-weight: var(--fw-semibold); color: var(--success); background: var(--success-soft); padding: 0.1875rem 0.5rem; border-radius: 6px; }
.loading-state, .empty-state { padding: 3rem; display: flex; flex-direction: column; align-items: center; justify-content: center; text-align: center; background: var(--surface-2); border: 1px solid var(--border); border-radius: 12px; }
.spinner { width: 2rem; height: 2rem; border: 3px solid var(--border); border-top-color: var(--brand); border-radius: 50%; animation: spin 0.8s linear infinite; margin-bottom: 0.75rem; }
@keyframes spin { to { transform: rotate(360deg); } }
.empty-title { margin-top: 1rem; font-size: var(--fs-xl); font-weight: var(--fw-medium); color: var(--text-strong); }
.empty-description { margin-top: 0.25rem; font-size: var(--fs-md); color: var(--text-weak); max-width: 24rem; }
.pixel-name { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: var(--fs-sm); background: var(--surface-soft); padding: 0.0625rem 0.375rem; border-radius: 4px; color: var(--text-2); }
.text-weak { color: var(--text-weak); font-size: var(--fs-base); }
.text-strong { color: var(--text-strong); font-weight: var(--fw-semibold); }
</style>
