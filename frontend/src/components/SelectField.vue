<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { Check, ChevronDown } from '@lucide/vue'

const props = defineProps<{
  id: string
  label: string
  modelValue: string
  options: { value: string; label: string; description?: string }[]
  disabled?: boolean
  error?: string
}>()
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
const root = ref<HTMLElement | null>(null)
const trigger = ref<HTMLButtonElement | null>(null)
const opened = ref(false)
const active = ref(0)
const selected = computed(() => props.options.find(option => option.value === props.modelValue))
function close(): void { opened.value = false }
async function open(): Promise<void> {
  if (props.disabled) return
  active.value = Math.max(0, props.options.findIndex(option => option.value === props.modelValue))
  opened.value = true
  await nextTick()
}
function choose(index: number): void {
  const option = props.options[index]
  if (option) emit('update:modelValue', option.value)
  close()
  trigger.value?.focus()
}
function keyboard(event: KeyboardEvent): void {
  if (event.key === 'Tab') { close(); return }
  if (event.key === 'Escape' && opened.value) { event.preventDefault(); event.stopPropagation(); close(); return }
  if (['ArrowDown', 'ArrowUp', 'Home', 'End', 'Enter', ' '].includes(event.key)) {
    event.preventDefault()
    if (!opened.value) { void open(); return }
    if (event.key === 'Enter' || event.key === ' ') { choose(active.value); return }
    if (event.key === 'Home') active.value = 0
    else if (event.key === 'End') active.value = props.options.length - 1
    else active.value = (active.value + (event.key === 'ArrowDown' ? 1 : -1) + props.options.length) % props.options.length
  } else if (event.key.length === 1) {
    const index = props.options.findIndex(option => option.label.toLowerCase().startsWith(event.key.toLowerCase()))
    if (index >= 0) { if (!opened.value) void open(); active.value = index }
  }
}
function outside(event: PointerEvent): void {
  if (!root.value?.contains(event.target as Node)) close()
}
onMounted(() => document.addEventListener('pointerdown', outside))
onBeforeUnmount(() => document.removeEventListener('pointerdown', outside))
</script>

<template>
  <div ref="root" class="select-field" @focusout="!root?.contains($event.relatedTarget as Node) && close()">
    <label :id="id + '-label'" :for="id" class="text-sm font-medium">{{ label }}</label>
    <button :id="id" ref="trigger" type="button" class="select-field-trigger" role="combobox" aria-haspopup="listbox"
      :aria-labelledby="id + '-label'" :aria-expanded="opened" :aria-controls="id + '-options'"
      :aria-activedescendant="opened ? id + '-option-' + active : undefined"
      :aria-invalid="error ? true : undefined" :aria-describedby="error ? id + '-error' : undefined"
      :disabled="disabled" @click="opened ? close() : open()" @keydown="keyboard">
      <span>{{ selected?.label }}</span><ChevronDown :size="16" aria-hidden="true" />
    </button>
    <div v-if="opened" :id="id + '-options'" class="select-field-options" role="listbox" :aria-labelledby="id + '-label'">
      <div v-for="(option, index) in options" :id="id + '-option-' + index" :key="option.value" role="option"
        :aria-selected="option.value === modelValue" class="select-field-option" :class="{ active: index === active }"
        @pointerdown.prevent @click="choose(index)" @pointermove="active = index">
        <span><strong>{{ option.label }}</strong><small v-if="option.description">{{ option.description }}</small></span>
        <Check v-if="option.value === modelValue" :size="16" aria-hidden="true" />
      </div>
    </div>
    <p v-if="error" :id="id + '-error'" class="text-sm text-vigil-down" role="alert">{{ error }}</p>
  </div>
</template>
