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
  <div class="bg-surface-soft border border-border rounded-xl overflow-hidden shadow-sm">
    <!-- Tabs Header -->
    <div class="flex items-center justify-between px-2 py-2 bg-surface border-b border-border">
      <div class="flex gap-1 overflow-x-auto">
        <button
          v-for="(tab, index) in tabs"
          :key="index"
          @click="activeTab = index"
          class="px-3 py-1.5 text-xs font-medium rounded-lg transition-colors whitespace-nowrap flex items-center gap-2"
          :class="[
            activeTab === index 
              ? 'bg-brand-soft text-brand' 
              : 'text-text-3 hover:text-text-1 hover:bg-surface-hover'
          ]"
        >
          <span v-if="tab.icon" class="text-base leading-none">{{ tab.icon }}</span>
          {{ tab.name }}
        </button>
      </div>
      <button 
        @click="copyCode"
        class="ml-2 p-1.5 text-text-3 hover:text-text-1 hover:bg-surface-hover rounded-lg transition-colors flex-shrink-0"
        title="Copiar código"
      >
        <Check v-if="copied" class="w-4 h-4 text-success" />
        <Clipboard v-else class="w-4 h-4" />
      </button>
    </div>
    
    <!-- Code Content -->
    <div class="p-4 bg-[#0d1117] overflow-x-auto text-sm font-mono leading-relaxed">
      <pre><code class="text-gray-300">{{ tabs[activeTab]?.code }}</code></pre>
    </div>
  </div>
</template>
