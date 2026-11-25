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
          class="block h-6 w-2.5 rounded-sm"
          :class="check.up ? 'bg-vigil-up' : 'bg-vigil-down'"
          :aria-label="checkLabel(check)"
          role="img"
        />
      </li>
    </ol>
    <p v-else class="text-sm text-vigil-muted">No checks yet.</p>
  </div>
</template>
