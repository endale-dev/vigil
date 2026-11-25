<script setup lang="ts">
import { computed } from 'vue'
import { CircleCheck, CircleHelp, CircleX, Pause, TriangleAlert } from '@lucide/vue'
import type { MonitorStatus, OverallStatus } from '../api/types'

const props = defineProps<{
  status: MonitorStatus | OverallStatus | 'paused'
  plain?: boolean
}>()

const presentation = computed(() => {
  switch (props.status) {
    case 'paused':
      return { label: 'Paused', icon: Pause, classes: 'text-vigil-muted' }
    case 'up':
      return { label: 'Up', icon: CircleCheck, classes: 'border-vigil-up/25 bg-vigil-up-soft text-vigil-up' }
    case 'operational':
      return { label: 'Operational', icon: CircleCheck, classes: 'border-vigil-up/25 bg-vigil-up-soft text-vigil-up' }
    case 'down':
      return { label: 'Down', icon: CircleX, classes: 'border-vigil-down/25 bg-vigil-down-soft text-vigil-down' }
    case 'degraded':
      return { label: 'Degraded', icon: TriangleAlert, classes: 'border-vigil-warn/25 bg-vigil-warn-soft text-vigil-warn' }
    default:
      return { label: 'Unknown', icon: CircleHelp, classes: 'border-vigil-unknown/20 bg-vigil-unknown/10 text-vigil-unknown' }
  }
})
</script>

<template>
  <span v-if="plain" class="monitor-state" :data-state="status" role="status">{{ presentation.label }}</span>
  <span v-else
    class="inline-flex items-center gap-1.5 rounded-md border px-2.5 py-1 text-xs font-semibold"
    :class="presentation.classes"
    role="status"
    :aria-label="`Status: ${presentation.label}`"
  >
    <component :is="presentation.icon" :size="14" :stroke-width="2.1" aria-hidden="true" />
    <span>{{ presentation.label }}</span>
  </span>
</template>
