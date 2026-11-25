<script setup lang="ts">
type FieldType = 'text' | 'email' | 'password' | 'url' | 'number'

withDefaults(defineProps<{
  id: string
  label: string
  modelValue: string
  type?: FieldType
  autocomplete?: string
  placeholder?: string
  required?: boolean
  disabled?: boolean
  error?: string
  min?: number
  max?: number
  step?: number
  maxLength?: number
}>(), {
  type: 'text',
  required: false,
  disabled: false,
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

function updateValue(event: Event): void {
  emit('update:modelValue', (event.target as HTMLInputElement).value)
}
</script>

<template>
  <div class="grid gap-1.5">
    <label class="text-sm font-medium" :for="id">{{ label }}</label>
    <input
      :id="id"
      :type="type"
      :value="modelValue"
      :autocomplete="autocomplete"
      :placeholder="placeholder"
      :required="required"
      :disabled="disabled"
      :min="min"
      :max="max"
      :step="step"
      :maxlength="maxLength"
      :aria-invalid="error ? 'true' : undefined"
      :aria-describedby="error ? `${id}-error` : undefined"
      class="min-h-10 rounded-md border border-vigil-rule bg-vigil-surface px-3 py-2 text-vigil-ink outline-none transition-shadow placeholder:text-vigil-muted/75 focus-visible:ring-2 focus-visible:ring-vigil-brand disabled:cursor-not-allowed disabled:opacity-60"
      @input="updateValue"
    />
    <p v-if="error" :id="`${id}-error`" class="text-sm text-vigil-down">
      {{ error }}
    </p>
  </div>
</template>
