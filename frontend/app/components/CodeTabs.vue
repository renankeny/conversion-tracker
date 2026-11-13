<script setup lang="ts">
import { ref } from 'vue'
import { Clipboard, Check } from 'lucide-vue-next'

const props = defineProps<{
  tabs: { name: string; icon?: string; code: string }[]
}>()

const activeTab = ref(0)
const copied = ref(false)

const copyCode = async () => {
  const code = props.tabs[activeTab.value]?.code
  if (code) {
    await navigator.clipboard.writeText(code)
    copied.value = true
    setTimeout(() => {
      copied.value = false
    }, 2000)
  }
}
</script>

<template>
  <div class="code-tabs-card">
    <div class="code-tabs-header">
      <div class="flex gap-1 overflow-x-auto">
        <button
          v-for="(tab, index) in tabs"
          :key="index"
          type="button"
          class="code-tab-btn"
          :class="{ 'is-active': activeTab === index }"
          @click="activeTab = index"
        >
          <span v-if="tab.icon" class="code-tab-icon">{{ tab.icon }}</span>
          {{ tab.name }}
        </button>
      </div>
      <button
        type="button"
        class="copy-btn"
        title="Copiar código"
        @click="copyCode"
      >
        <Check v-if="copied" class="w-4 h-4 text-success" />
        <Clipboard v-else class="w-4 h-4" />
      </button>
    </div>

    <div class="code-body">
      <pre><code class="code-text">{{ tabs[activeTab]?.code }}</code></pre>
    </div>
  </div>
</template>

<style scoped>
.code-tabs-card { background: var(--surface-soft); border: 1px solid var(--border); border-radius: 12px; overflow: hidden; }
.code-tabs-header { display: flex; align-items: center; justify-content: space-between; padding: 0.5rem; background: var(--surface-2); border-bottom: 1px solid var(--border); }
.code-tab-btn { padding: 0.375rem 0.75rem; font-size: var(--fs-base); font-weight: var(--fw-medium); border-radius: 8px; white-space: nowrap; display: flex; align-items: center; gap: 0.5rem; color: var(--text-3); background: transparent; border: none; cursor: pointer; transition: all 0.15s ease; }
.code-tab-btn:hover { color: var(--text-1); background: var(--surface-soft); }
.code-tab-btn.is-active { background: var(--brand-soft); color: var(--brand); }
.code-tab-icon { font-size: var(--fs-xl); line-height: 1; }
.copy-btn { margin-left: 0.5rem; padding: 0.375rem; color: var(--text-3); background: transparent; border: none; border-radius: 8px; cursor: pointer; flex-shrink: 0; transition: all 0.15s ease; }
.copy-btn:hover { color: var(--text-1); background: var(--surface-soft); }
.code-body { padding: 1rem; background: var(--surface-code); overflow-x: auto; font-family: var(--font-mono); font-size: var(--fs-lg); line-height: 1.6; }
.code-text { color: var(--text-2); }
</style>
