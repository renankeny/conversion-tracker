<template>
  <div class="advanced-mapping-section">
    <div class="right-pane">
      <label>Mapeamento de Campos (JSONPath)</label>
      <p class="help-text">Ex: `data.customer.email` ou `transaction.amount`</p>

      <datalist id="jsonpaths">
        <option v-for="path in availableJsonPaths" :key="path" :value="path" />
      </datalist>

      <div class="mapping-grid">
        <div v-for="f in fieldDefs" :key="f.key" class="form-group">
          <label>{{ f.label }}</label>
          <input
            v-model="mappingConfig[f.key]"
            type="text"
            list="jsonpaths"
            class="base-input"
            :placeholder="f.placeholder"
          />
          <p v-if="exampleJson && simulatedResult[f.key]" class="simulated-value">
            ↳ <strong>{{ simulatedResult[f.key] }}</strong>
          </p>
        </div>
      </div>

      <h4>Regras de Status (Status Map)</h4>
      <p class="help-text">Quais valores indicam Venda (Purchase) ou Boleto (InitiateCheckout)? Ex: paid -&gt; Purchase</p>

      <div v-for="(_, key) in mappingConfig.status_map" :key="key" class="status-row">
        <input type="text" :value="key" class="base-input flex-1" @change="updateStatusKey(String(key), $event)" />
        <span>-&gt;</span>
        <div class="flex-1">
          <BaseDropdown
            v-model="mappingConfig.status_map[key]"
            :options="[
              { value: 'Purchase', label: 'Purchase (Venda Aprovada)' },
              { value: 'InitiateCheckout', label: 'InitiateCheckout (Boleto/Pix)' }
            ]"
          />
        </div>
        <button type="button" class="btn-remove" @click="removeStatusMap(String(key))">X</button>
      </div>
      <button type="button" class="btn-add" @click="addStatusMap()">+ Adicionar Regra</button>
    </div>
  </div>
</template>

<script setup lang="ts">
const props = defineProps<{
  mappingConfig: Record<string, any>
  exampleJson: string
}>()

const fieldDefs = [
  { key: 'first_name', label: 'Primeiro Nome', placeholder: 'customer.first_name' },
  { key: 'last_name', label: 'Sobrenome', placeholder: 'customer.last_name' },
  { key: 'email', label: 'E-mail', placeholder: 'customer.email' },
  { key: 'phone', label: 'Telefone', placeholder: 'customer.phone' },
  { key: 'transaction_id', label: 'ID da Transação', placeholder: 'transaction.id' },
  { key: 'amount', label: 'Valor da Compra', placeholder: 'transaction.amount' },
  { key: 'status', label: 'Status (Campo)', placeholder: 'status' },
  { key: 'utm_source', label: 'UTM Source (Opcional)', placeholder: 'metadata.utm_source' },
]

const availableJsonPaths = computed(() => {
  if (!props.exampleJson) return []
  try {
    const parsed = JSON.parse(props.exampleJson)
    const paths: string[] = []
    const flatten = (obj: any, prefix = '') => {
      if (obj === null || typeof obj !== 'object') {
        if (prefix) paths.push(prefix)
        return
      }
      if (Array.isArray(obj)) {
        if (obj.length > 0) flatten(obj[0], prefix + '[0]')
      } else {
        for (const k in obj) flatten(obj[k], prefix ? `${prefix}.${k}` : k)
      }
    }
    flatten(parsed)
    return paths
  } catch {
    return []
  }
})

const extractValue = (obj: any, path: string) => {
  if (!obj || !path) return ''
  try {
    const parts = path.replace(/\[(\w+)\]/g, '.$1').split('.').filter(Boolean)
    let cur = obj
    for (const part of parts) {
      if (cur == null) return ''
      cur = cur[part]
    }
    return cur
  } catch {
    return ''
  }
}

const simulatedResult = computed<Record<string, any>>(() => {
  if (!props.exampleJson) return {}
  try {
    const parsed = JSON.parse(props.exampleJson)
    const conf = props.mappingConfig
    let fn = String(extractValue(parsed, conf.first_name) || '')
    let ln = String(extractValue(parsed, conf.last_name) || '')
    if (fn && (fn === ln || !ln)) {
      const parts = fn.trim().split(/\s+/)
      if (parts.length > 1) {
        fn = parts[0] || ''
        ln = parts.slice(1).join(' ')
      } else if (fn === ln) {
        ln = ''
      }
    }
    const statusRaw = String(extractValue(parsed, conf.status) || '')
    const statusMapped = conf.status_map?.[statusRaw] || statusRaw
    return {
      first_name: fn,
      last_name: ln,
      email: extractValue(parsed, conf.email),
      phone: extractValue(parsed, conf.phone),
      transaction_id: extractValue(parsed, conf.transaction_id),
      amount: extractValue(parsed, conf.amount),
      status: statusRaw ? `${statusRaw} → ${statusMapped}` : '',
      utm_source: extractValue(parsed, conf.utm_source),
    }
  } catch {
    return {}
  }
})

const updateStatusKey = (oldKey: string, e: Event) => {
  const newKey = (e.target as HTMLInputElement).value
  if (oldKey === newKey || !newKey) return
  const val = props.mappingConfig.status_map[oldKey]
  delete props.mappingConfig.status_map[oldKey]
  props.mappingConfig.status_map[newKey] = val
}

const removeStatusMap = (key: string) => {
  delete props.mappingConfig.status_map[key]
}

const addStatusMap = () => {
  props.mappingConfig.status_map['novo_status'] = 'Purchase'
}
</script>

<style scoped>
.advanced-mapping-section { display: flex; flex-direction: column; gap: 1rem; margin-top: 1rem; }
.right-pane { display: flex; flex-direction: column; gap: 1rem; }
.mapping-grid { display: grid; grid-template-columns: 1fr; gap: 1rem; }
@media (min-width: 640px) { .mapping-grid { grid-template-columns: 1fr 1fr; } }
.base-input, .base-select { width: 100%; padding: 0.5rem; border-radius: 6px; border: 1px solid var(--border); background: var(--surface-2); color: var(--text-1); }
.help-text { font-size: var(--fs-sm); color: var(--text-3); margin-top: -0.5rem; margin-bottom: 0.5rem; }
.form-group { display: flex; flex-direction: column; gap: 0.25rem; }
.form-group label { font-size: var(--fs-md); font-weight: var(--fw-medium); }
.simulated-value { margin-top: 0.35rem; font-size: var(--fs-sm); color: var(--brand); padding: 0.25rem 0.5rem; background: var(--brand-soft); border-radius: 4px; display: inline-block; }
.simulated-value strong { font-weight: var(--fw-bold); }
.status-row { display: flex; align-items: center; gap: 0.5rem; margin-bottom: 0.5rem; flex-wrap: wrap; }
.btn-remove { background: none; border: none; color: var(--danger); cursor: pointer; }
.btn-add { background: none; border: 1px dashed var(--border); color: var(--text-2); padding: 0.5rem; border-radius: 6px; cursor: pointer; }
</style>
