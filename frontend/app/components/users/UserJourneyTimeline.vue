<template>
  <div class="user-journey-timeline">
    <div v-if="loadingTimeline" class="muted py-4">Carregando jornada...</div>
    <div v-else-if="!timelineEvents.length" class="empty-hint-card">Nenhum evento adicional registrado.</div>
    <div v-else class="journey-timeline">
      <div v-for="(item, idx) in timelineEvents" :key="idx" class="timeline-node">
        <div class="timeline-indicator" :class="item.typeClass">
          <span class="timeline-dot" />
          <div v-if="idx < timelineEvents.length - 1" class="timeline-line" />
        </div>
        <div class="timeline-body">
          <div class="timeline-header">
            <div class="flex items-center gap-2">
              <span class="timeline-badge" :class="item.typeClass">{{ item.badge }}</span>
              <span class="timeline-title">{{ item.title }}</span>
            </div>
            <span class="timeline-date">{{ item.date }}</span>
          </div>
          <p v-if="item.description" class="timeline-desc">{{ item.description }}</p>
          <div v-if="item.meta && item.meta.length" class="timeline-meta">
            <span v-for="(m, mIdx) in item.meta" :key="mIdx" class="timeline-chip">
              <strong>{{ m.label }}:</strong> {{ m.value }}
            </span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
export interface TimelineItem {
  badge: string
  title: string
  description?: string
  date: string
  typeClass: string
  meta?: Array<{ label: string; value: string }>
}

interface Props {
  timelineEvents: TimelineItem[]
  loadingTimeline: boolean
}

defineProps<Props>()
</script>

<style scoped>
.user-journey-timeline {
  display: flex;
  flex-direction: column;
}

.empty-hint-card {
  font-size: var(--fs-sm);
  color: var(--text-3);
  font-style: italic;
  padding: 1.25rem;
  text-align: center;
  background: var(--surface-soft-2);
  border: 1px dashed var(--border-soft);
  border-radius: 8px;
}

.muted {
  color: var(--text-3);
  font-style: italic;
  font-size: var(--fs-sm);
}

.journey-timeline {
  display: flex;
  flex-direction: column;
  position: relative;
  max-height: 320px;
  overflow-y: auto;
  padding-right: 0.5rem;
}

.timeline-node {
  display: flex;
  gap: 0.75rem;
  position: relative;
  padding-bottom: 0.875rem;
}

.timeline-node:last-child {
  padding-bottom: 0.25rem;
}

.timeline-indicator {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 16px;
  position: relative;
  padding-top: 6px;
  flex-shrink: 0;
}

.timeline-dot {
  width: 10px;
  height: 10px;
  border-radius: 9999px;
  background: var(--text-3);
  border: 2px solid var(--surface-1, #18181b);
  box-shadow: 0 0 0 1px var(--border);
  z-index: 2;
  flex-shrink: 0;
}

.timeline-indicator.type-discovery .timeline-dot { background: #38bdf8; box-shadow: 0 0 8px rgba(56, 189, 248, 0.4); }
.timeline-indicator.type-checkout .timeline-dot { background: #fbbf24; box-shadow: 0 0 8px rgba(251, 191, 36, 0.4); }
.timeline-indicator.type-purchase .timeline-dot { background: var(--brand); box-shadow: 0 0 8px var(--brand-border); }
.timeline-indicator.type-lead .timeline-dot { background: #a855f7; box-shadow: 0 0 8px rgba(168, 85, 247, 0.4); }
.timeline-indicator.type-contact .timeline-dot { background: var(--brand); box-shadow: 0 0 8px var(--brand-border); }
.timeline-indicator.type-device .timeline-dot { background: #ec4899; box-shadow: 0 0 8px rgba(236, 72, 153, 0.4); }

.timeline-line {
  position: absolute;
  top: 16px;
  bottom: -4px;
  left: 50%;
  transform: translateX(-50%);
  width: 2px;
  background: var(--border-soft);
  z-index: 1;
}

.timeline-body {
  flex: 1;
  background: var(--surface-soft-2);
  border: 1px solid var(--border-soft);
  border-radius: 8px;
  padding: 0.5rem 0.75rem;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.timeline-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}

.timeline-badge {
  font-size: var(--fs-2xs);
  font-weight: var(--fw-bold);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  padding: 0.125rem 0.375rem;
  border-radius: 4px;
  background: var(--surface-soft);
  color: var(--text-2);
}

.timeline-badge.type-discovery { background: rgba(56, 189, 248, 0.15); color: #38bdf8; border: 1px solid rgba(56, 189, 248, 0.25); }
.timeline-badge.type-checkout { background: rgba(251, 191, 36, 0.15); color: #fbbf24; border: 1px solid rgba(251, 191, 36, 0.25); }
.timeline-badge.type-purchase { background: var(--brand-soft); color: var(--brand); border: 1px solid var(--brand-border); }
.timeline-badge.type-lead { background: rgba(168, 85, 247, 0.15); color: #c084fc; border: 1px solid rgba(168, 85, 247, 0.25); }
.timeline-badge.type-contact { background: var(--brand-soft); color: var(--brand); border: 1px solid var(--brand-border); }
.timeline-badge.type-device { background: rgba(236, 72, 153, 0.15); color: #f472b6; border: 1px solid rgba(236, 72, 153, 0.25); }

.timeline-title {
  font-size: var(--fs-base);
  font-weight: var(--fw-semibold);
  color: var(--text-1);
}

.timeline-date {
  font-size: var(--fs-xs);
  color: var(--text-3);
  white-space: nowrap;
}

.timeline-desc {
  font-size: var(--fs-sm);
  color: var(--text-2);
  margin: 0;
}

.timeline-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 0.375rem;
  margin-top: 0.25rem;
}

.timeline-chip {
  font-size: var(--fs-xs);
  background: var(--surface-deep);
  border: 1px solid var(--border-soft);
  padding: 0.125rem 0.375rem;
  border-radius: 4px;
  color: var(--text-2);
}

.timeline-chip strong {
  color: var(--text-1);
}
</style>
