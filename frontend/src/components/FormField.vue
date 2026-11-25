<script setup lang="ts">
import { ref } from 'vue'
import { Eye, EyeOff } from '@lucide/vue'

type FieldType = 'text' | 'email' | 'password' | 'url' | 'number'

withDefaults(defineProps<{
  id: string
  label: string
  modelValue: string
  type?: FieldType
  inputmode?: 'text' | 'url'
  autofocus?: boolean
  autocomplete?: string
  placeholder?: string
  required?: boolean
  disabled?: boolean
  error?: string
  min?: number
  max?: number
  step?: number
  maxLength?: number
  hint?: string
  revealable?: boolean
}>(), {
  type: 'text',
  required: false,
  disabled: false,
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const revealed = ref(false)

function updateValue(event: Event): void {
  emit('update:modelValue', (event.target as HTMLInputElement).value)
}
</script>

<template>
  <div class="grid gap-1.5">
    <div class="flex items-baseline justify-between gap-2">
      <label class="text-sm font-medium" :for="id">{{ label }}</label>
      <slot name="label-action" />
    </div>
    <div class="relative grid min-w-0">
      <input
        :id="id"
        :autofocus="autofocus"
        :type="revealable && revealed ? 'text' : type"
        :inputmode="inputmode"
        :spellcheck="type === 'password' || inputmode === 'url' ? false : undefined"
        :autocapitalize="type === 'password' || type === 'email' || inputmode === 'url' ? 'none' : undefined"
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
        :aria-describedby="[hint && id + '-hint', error && id + '-error'].filter(Boolean).join(' ') || undefined"
        :class="{ 'pr-12': revealable }"
        class="form-input min-h-10 rounded-md border border-vigil-rule bg-vigil-surface px-3 py-2 text-vigil-ink placeholder:text-vigil-muted/75 disabled:cursor-not-allowed disabled:opacity-60"
        @input="updateValue"
      />
      <button v-if="revealable && type === 'password'" type="button" class="field-reveal" :aria-controls="id"
        :aria-label="`${revealed ? 'Hide' : 'Show'} ${label.toLowerCase()}`" :disabled="disabled" @click="revealed = !revealed">
        <component :is="revealed ? EyeOff : Eye" :size="17" aria-hidden="true" />
      </button>
    </div>
    <p v-if="hint" :id="id + '-hint'" class="text-xs text-vigil-muted">{{ hint }}</p>
    <p v-if="error" :id="`${id}-error`" class="text-sm text-vigil-down" role="alert">
      {{ error }}
    </p>
  </div>
</template>

<style scoped>
.form-input { transition: border-color .15s, box-shadow .15s; }
.form-input:hover:not(:disabled) { border-color: #91aa9a; }
.form-input:focus { outline: none; border-color: #397d63; box-shadow: 0 0 0 3px #17624d1a; }
.form-input[aria-invalid=true] { border-color: #ae3939; }
.form-input[aria-invalid=true]:focus { box-shadow: 0 0 0 3px #ae39391a; }
</style>
