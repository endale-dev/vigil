<script setup lang="ts">
import { computed } from 'vue'
import { CircleCheck, CircleHelp, CircleX, TriangleAlert } from '@lucide/vue'
import type { MonitorStatus, OverallStatus } from '../api/types'

const props = defineProps<{
  status: MonitorStatus | OverallStatus
  tone?: 'dark' | 'light'
}>()

const presentation = computed(() => {
  const light = props.tone === 'light'
  switch (props.status) {
    case 'up':
      return { label: 'Up', icon: CircleCheck, classes: light ? 'border-emerald-200 bg-emerald-50 text-emerald-800' : 'border-emerald-700 bg-emerald-950 text-emerald-200' }
    case 'operational':
      return { label: 'Operational', icon: CircleCheck, classes: light ? 'border-emerald-200 bg-emerald-50 text-emerald-800' : 'border-emerald-700 bg-emerald-950 text-emerald-200' }
    case 'down':
      return { label: 'Down', icon: CircleX, classes: light ? 'border-red-200 bg-red-50 text-red-800' : 'border-red-700 bg-red-950 text-red-200' }
    case 'degraded':
      return { label: 'Degraded', icon: TriangleAlert, classes: light ? 'border-amber-200 bg-amber-50 text-amber-800' : 'border-amber-700 bg-amber-950 text-amber-200' }
    default:
      return { label: 'Unknown', icon: CircleHelp, classes: light ? 'border-slate-200 bg-slate-100 text-slate-700' : 'border-slate-600 bg-slate-900 text-slate-200' }
  }
})
</script>

<template>
  <span
    class="inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs font-semibold"
    :class="presentation.classes"
    role="status"
    :aria-label="`Status: ${presentation.label}`"
  >
    <component :is="presentation.icon" :size="14" :stroke-width="2.1" aria-hidden="true" />
    <span>{{ presentation.label }}</span>
  </span>
</template>
