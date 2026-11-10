<template>
  <div class="wa-audio">
    <audio
      ref="audioEl"
      :src="src"
      preload="metadata"
      @timeupdate="onTimeUpdate"
      @loadedmetadata="onLoadedMeta"
      @ended="playing = false"
    />

    <button type="button" class="play-btn" :aria-label="playing ? 'Pausar' : 'Ouvir'" @click="togglePlay">
      <Pause v-if="playing" :size="14" />
      <Play v-else :size="14" />
    </button>

    <div class="track-col">
      <div class="progress-bar" @click="seek">
        <div class="progress-fill" :style="{ width: `${progressPct}%` }" />
      </div>
      <span class="time-label">{{ formattedCurrent }} / {{ formattedDuration }}</span>
    </div>

    <button type="button" class="speed-btn" @click="cycleRate">
      {{ playbackRate }}x
    </button>
  </div>
</template>

<script setup lang="ts">
import { Play, Pause } from 'lucide-vue-next'

const props = defineProps<{
  src: string
  initialDuration?: number
}>()

const audioEl = ref<HTMLAudioElement | null>(null)
const playing = ref(false)
const currentSec = ref(0)
const durationSec = ref(props.initialDuration || 0)
const playbackRate = ref<1 | 1.5 | 2>(1)

const progressPct = computed(() => {
  if (!durationSec.value) return 0
  return Math.min(100, (currentSec.value / durationSec.value) * 100)
})

const fmt = (sec: number): string => {
  const total = Math.max(0, Math.floor(sec || 0))
  const m = Math.floor(total / 60)
  const s = total % 60
  return `${m}:${String(s).padStart(2, '0')}`
}

const formattedCurrent = computed(() => fmt(currentSec.value))
const formattedDuration = computed(() => fmt(durationSec.value))

const togglePlay = async () => {
  const el = audioEl.value
  if (!el) return
  if (playing.value) {
    el.pause()
    playing.value = false
  } else {
    try {
      el.playbackRate = playbackRate.value
      await el.play()
      playing.value = true
    } catch {
      playing.value = false
    }
  }
}

const onTimeUpdate = () => {
  if (audioEl.value) currentSec.value = audioEl.value.currentTime || 0
}

const onLoadedMeta = () => {
  const d = audioEl.value?.duration
  if (d && Number.isFinite(d) && d > 0) durationSec.value = d
}

const seek = (ev: MouseEvent) => {
  const el = audioEl.value
  if (!el || !durationSec.value) return
  const rect = (ev.currentTarget as HTMLElement).getBoundingClientRect()
  const ratio = Math.max(0, Math.min(1, (ev.clientX - rect.left) / rect.width))
  el.currentTime = ratio * durationSec.value
  currentSec.value = el.currentTime
}

const cycleRate = () => {
  const next = playbackRate.value === 1 ? 1.5 : playbackRate.value === 1.5 ? 2 : 1
  playbackRate.value = next
  if (audioEl.value) audioEl.value.playbackRate = next
}

onUnmounted(() => {
  audioEl.value?.pause()
})
</script>

<style scoped>
.wa-audio {
  display: flex;
  align-items: center;
  gap: 0.625rem;
  min-width: 210px;
  max-width: 280px;
  padding: 0.25rem 0;
}
.play-btn {
  width: 30px; height: 30px; border-radius: 50%; border: none;
  background: var(--brand); color: var(--brand-text);
  display: flex; align-items: center; justify-content: center;
  cursor: pointer; flex-shrink: 0;
}
.track-col { flex: 1; display: flex; flex-direction: column; gap: 0.25rem; min-width: 0; }
.progress-bar {
  height: 5px; border-radius: 999px; background: var(--border);
  overflow: hidden; cursor: pointer;
}
.progress-fill { height: 100%; background: var(--brand); border-radius: 999px; transition: width 0.1s linear; }
.time-label { font-family: var(--font-mono); font-size: var(--fs-2xs); color: var(--text-3); }
.speed-btn {
  padding: 0.15rem 0.4rem; border-radius: 6px;
  background: var(--surface-deep); border: 1px solid var(--border);
  color: var(--text-2); font-size: var(--fs-2xs); font-weight: var(--fw-semibold);
  cursor: pointer; flex-shrink: 0;
}
</style>

