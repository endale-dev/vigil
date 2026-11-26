<script setup lang="ts" generic="T">
import { computed, ref, watch } from 'vue'
import { ChevronLeft, ChevronRight } from '@lucide/vue'

const props = defineProps<{
  items: readonly T[]
  pageSize: number
  label: string
  resetOn?: readonly unknown[]
}>()

const page = ref(1)
const pageCount = computed(() => Math.max(1, Math.ceil(props.items.length / props.pageSize)))
const start = computed(() => (page.value - 1) * props.pageSize)
const end = computed(() => Math.min(start.value + props.pageSize, props.items.length))
const visibleItems = computed(() => props.items.slice(start.value, end.value))

watch(pageCount, count => { page.value = Math.min(page.value, count) })
watch(() => props.resetOn, () => { page.value = 1 })
</script>

<template>
  <slot :items="visibleItems" />
  <div v-if="pageCount > 1 || $slots.footer" class="list-footer">
    <nav v-if="pageCount > 1" class="list-pagination" :aria-label="label + ' pagination'">
      <span role="status">{{ start + 1 }}–{{ end }} of {{ items.length }} {{ label }}</span>
      <button type="button" class="icon-control" :aria-label="'Previous page of ' + label" :disabled="page === 1" @click="page--">
        <ChevronLeft :size="16" aria-hidden="true" />
      </button>
      <button type="button" class="icon-control" :aria-label="'Next page of ' + label" :disabled="page >= pageCount" @click="page++">
        <ChevronRight :size="16" aria-hidden="true" />
      </button>
    </nav>
    <slot name="footer" />
  </div>
</template>
