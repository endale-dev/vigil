<script setup lang="ts">
import { computed } from 'vue'
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
    <ol v-if="chronologicalChecks.length" class="flex flex-wrap gap-1" aria-label="Recent monitor checks">
      <li v-for="check in chronologicalChecks" :key="check.id" :title="checkLabel(check)">
        <span
          class="flex size-7 items-center justify-center rounded-sm text-xs font-semibold"
          :class="check.up ? 'bg-emerald-950 text-emerald-200' : 'bg-red-950 text-red-200'"
          :aria-label="checkLabel(check)"
          role="img"
        >
          <span aria-hidden="true">{{ check.up ? '✓' : '×' }}</span>
        </span>
      </li>
    </ol>
    <p v-else class="text-sm text-slate-400">No checks yet.</p>
  </div>
</template>
