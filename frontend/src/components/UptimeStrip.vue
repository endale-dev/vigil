<script setup lang="ts">
import { computed } from 'vue'
import { Activity } from '@lucide/vue'
import type { CheckDTO } from '../api/types'

const props = defineProps<{
  checks: CheckDTO[]
}>()

const chronologicalChecks = computed(() => [...props.checks].reverse())

function checkLabel(check: CheckDTO): string {
  const checkedAt = new Date(check.checkedAt)
  const time = Number.isNaN(checkedAt.getTime()) ? check.checkedAt : checkedAt.toLocaleString()
  return `${check.up ? 'Up' : 'Down'}, ${check.latencyMs} ms, checked ${time}`
}
</script>

<template>
  <div>
    <ol v-if="chronologicalChecks.length" class="flex gap-1" aria-label="Recent monitor checks, oldest to newest">
      <li v-for="check in chronologicalChecks" :key="check.id" class="min-w-0 max-w-6 flex-1" :title="checkLabel(check)">
        <span
          class="block h-6 rounded-sm"
          :class="check.up ? 'bg-vigil-up' : 'bg-vigil-down'"
          :aria-label="checkLabel(check)"
          role="img"
        />
      </li>
    </ol>
    <div v-else class="flex items-start gap-3 py-2">
      <span class="grid size-9 shrink-0 place-items-center rounded-md bg-vigil-brand-soft text-vigil-brand" aria-hidden="true">
        <Activity :size="17" :stroke-width="1.8" />
      </span>
      <div>
        <p class="text-sm font-medium text-vigil-ink">The first check is on its way</p>
        <p class="mt-1 text-sm text-vigil-muted">Recent results will appear here.</p>
      </div>
    </div>
  </div>
</template>
