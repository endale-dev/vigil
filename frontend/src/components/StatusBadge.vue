<script setup lang="ts">
import { computed } from 'vue'
import type { MonitorStatus, OverallStatus } from '../api/types'

const props = defineProps<{
  status: MonitorStatus | OverallStatus
}>()

const presentation = computed(() => {
  switch (props.status) {
    case 'up':
      return { label: 'Up', icon: '✓', classes: 'border-emerald-700 bg-emerald-950 text-emerald-200' }
    case 'operational':
      return { label: 'Operational', icon: '✓', classes: 'border-emerald-700 bg-emerald-950 text-emerald-200' }
    case 'down':
      return { label: 'Down', icon: '×', classes: 'border-red-700 bg-red-950 text-red-200' }
    case 'degraded':
      return { label: 'Degraded', icon: '!', classes: 'border-amber-700 bg-amber-950 text-amber-200' }
    default:
      return { label: 'Unknown', icon: '?', classes: 'border-slate-600 bg-slate-900 text-slate-200' }
  }
})
</script>

<template>
  <span
    class="inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs font-medium"
    :class="presentation.classes"
    role="status"
    :aria-label="`Status: ${presentation.label}`"
  >
    <span aria-hidden="true">{{ presentation.icon }}</span>
    <span>{{ presentation.label }}</span>
  </span>
</template>
