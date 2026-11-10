<template>
  <div class="brand-logo" :class="[`size-${size}`]">
    <!-- Imagem customizada quando configurada para exibir logo -->
    <img
      v-if="showCustomImage"
      :src="resolvedLogoUrl"
      :alt="resolvedName"
      class="brand-logo-img"
    />

    <!-- Ícone padrão quando modo é icon_text ou quando não há imagem cadastrada -->
    <div v-else class="brand-icon-box">
      <TrendingUp :size="iconSize" class="brand-icon-svg" />
    </div>

    <!-- Texto do nome da plataforma + subtítulo opcional -->
    <div v-if="showText" class="brand-text-col">
      <span class="brand-name-text">{{ resolvedName }}</span>
      <span v-if="showTagline && resolvedTagline" class="brand-tagline-text">
        {{ resolvedTagline }}
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { TrendingUp } from 'lucide-vue-next'
import type { LogoDisplayMode } from '#shared/types'

interface Props {
  size?: 'sm' | 'md' | 'lg'
  showTagline?: boolean
  overrideName?: string
  overrideTagline?: string
  overrideLogoUrl?: string
  overrideMode?: LogoDisplayMode
}

const props = withDefaults(defineProps<Props>(), {
  size: 'md',
  showTagline: false,
  overrideName: undefined,
  overrideTagline: undefined,
  overrideLogoUrl: undefined,
  overrideMode: undefined,
})

const { branding, brandName, tagline, activeLogoUrl } = useBranding()

const resolvedName = computed(() => props.overrideName ?? brandName.value)
const resolvedTagline = computed(() => props.overrideTagline ?? tagline.value)
const resolvedLogoUrl = computed(() => props.overrideLogoUrl ?? activeLogoUrl.value)
const resolvedMode = computed<LogoDisplayMode>(
  () => props.overrideMode ?? branding.value.logo_display_mode ?? 'icon_text'
)

const showCustomImage = computed(() => {
  return Boolean(resolvedLogoUrl.value?.trim())
})

const showText = computed(() => {
  if (resolvedMode.value === 'logo_only' && !!resolvedLogoUrl.value?.trim()) {
    return false
  }
  return true
})

const iconSize = computed(() => {
  if (props.size === 'sm') return 18
  if (props.size === 'lg') return 24
  return 20
})
</script>

<style scoped>
.brand-logo {
  display: inline-flex;
  align-items: center;
  gap: 0.625rem;
  text-decoration: none;
  color: var(--text-strong);
  min-width: 0;
}

.brand-logo-img {
  object-fit: contain;
  width: auto;
  flex-shrink: 0;
}

.size-sm .brand-logo-img {
  max-height: 28px;
  max-width: 140px;
}

.size-md .brand-logo-img {
  max-height: 36px;
  max-width: 170px;
}

.size-lg .brand-logo-img {
  max-height: 44px;
  max-width: 210px;
}

.brand-icon-box {
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  background: var(--brand-soft);
  border: 1px solid var(--brand-border);
  color: var(--brand);
  flex-shrink: 0;
}

.size-sm .brand-icon-box {
  width: 32px;
  height: 32px;
}

.size-md .brand-icon-box {
  width: 36px;
  height: 36px;
}

.size-lg .brand-icon-box {
  width: 44px;
  height: 44px;
  border-radius: 10px;
}

.brand-icon-svg {
  color: var(--brand);
}

.brand-text-col {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.brand-name-text {
  font-family: 'Instrument Sans', sans-serif;
  font-weight: var(--fw-bold);
  color: var(--text-strong);
  letter-spacing: -0.02em;
  line-height: 1.15;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.size-sm .brand-name-text {
  font-size: var(--fs-lg);
}

.size-md .brand-name-text {
  font-size: var(--fs-xl);
}

.size-lg .brand-name-text {
  font-size: var(--fs-2xl);
}

.brand-tagline-text {
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--brand);
  margin-top: 2px;
}
</style>

