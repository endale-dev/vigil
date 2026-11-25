<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { MonitorDTO, MonitorRequest, MonitorType } from '../api/types'
import FormField from './FormField.vue'

type FormState = Omit<MonitorRequest, 'intervalSeconds' | 'timeoutSeconds' | 'expectedStatus' | 'tlsWarnDays'> & {
  intervalSeconds: string
  timeoutSeconds: string
  expectedStatus: string
  tlsWarnDays: string
}

const props = withDefaults(defineProps<{
  initialValue?: MonitorDTO
  fieldErrors?: Record<string, string | undefined>
  disabled?: boolean
  submitLabel?: string
}>(), {
  disabled: false,
  submitLabel: 'Save monitor',
})

const emit = defineEmits<{
  submit: [value: MonitorRequest]
}>()

function initialForm(value?: MonitorDTO): FormState {
  const type = value?.type ?? 'http'
  return {
    name: value?.name ?? '',
    type,
    target: value?.target ?? '',
    intervalSeconds: String(value?.intervalSeconds ?? 60),
    timeoutSeconds: String(value?.timeoutSeconds ?? 10),
    expectedStatus: type === 'http' ? String(value?.expectedStatus ?? 200) : '0',
    tlsWarnDays: type === 'tls' ? String(value?.tlsWarnDays ?? 14) : '0',
    public: value?.public ?? false,
  }
}

const form = ref<FormState>(initialForm(props.initialValue))
const timeoutExceedsInterval = computed(() => Number(form.value.timeoutSeconds) > Number(form.value.intervalSeconds))
const validationError = ref('')
const intervalError = () => props.fieldErrors?.intervalSeconds ?? props.fieldErrors?.interval
const timeoutError = () => props.fieldErrors?.timeoutSeconds ?? props.fieldErrors?.timeout

watch(() => props.initialValue, (value) => {
  form.value = initialForm(value)
}, { deep: true })

function updateType(event: Event): void {
  const type = (event.target as HTMLSelectElement).value as MonitorType
  if (type !== 'http') {
    form.value.expectedStatus = '0'
  } else {
    form.value.expectedStatus = '200'
  }
  form.value.tlsWarnDays = type === 'tls' ? '14' : '0'
  form.value.type = type
  validationError.value = ''
}

function submit(): void {
  const intervalSeconds = Number(form.value.intervalSeconds)
  const timeoutSeconds = Number(form.value.timeoutSeconds)
  const expectedStatus = Number(form.value.expectedStatus)
  validationError.value = ''
  if (timeoutSeconds > intervalSeconds) {
    validationError.value = 'Timeout cannot be greater than the check interval.'
    return
  }
  if (form.value.type === 'http' && expectedStatus !== 0 && expectedStatus < 100) {
    validationError.value = 'Enter 0 or an HTTP status from 100 to 599.'
    return
  }

  emit('submit', {
    name: form.value.name,
    type: form.value.type,
    target: form.value.target,
    intervalSeconds,
    timeoutSeconds,
    expectedStatus: form.value.type === 'http' ? expectedStatus : 0,
    tlsWarnDays: form.value.type === 'tls' ? Number(form.value.tlsWarnDays) : 0,
    public: form.value.public,
  })
}
</script>

<template>
  <form class="grid gap-5" @submit.prevent="submit">
    <FormField
      id="monitor-name"

      :model-value="form.name"
      @update:model-value="form.name = $event"
      label="Monitor name"
      autocomplete="off"
      :max-length="80"
      :disabled="disabled"
      :error="fieldErrors?.name"
      required
    />

    <div class="grid gap-1.5">
      <label class="text-sm font-medium" for="monitor-type">Monitor type</label>
      <select
        id="monitor-type"
        :value="form.type"
        class="min-h-10 rounded-md border border-vigil-rule bg-vigil-surface px-3 py-2 text-vigil-ink outline-none focus-visible:ring-2 focus-visible:ring-vigil-brand disabled:cursor-not-allowed disabled:opacity-60"
        :disabled="disabled"
        @change="updateType"
      >
        <option value="http">HTTP</option>
        <option value="tcp">TCP</option>
        <option value="tls">TLS certificate</option>
      </select>
      <p v-if="fieldErrors?.type" class="text-sm text-vigil-down">{{ fieldErrors.type }}</p>
    </div>

    <FormField
      id="monitor-target"

      :model-value="form.target"
      @update:model-value="form.target = $event"
      :label="form.type === 'http' ? 'URL' : 'Host and port'"
      :type="form.type === 'http' ? 'url' : 'text'"
      :placeholder="form.type === 'http' ? 'https://example.com/health' : 'example.com:443'"
      :disabled="disabled"
      :error="fieldErrors?.target"
      required
    />

    <div class="grid gap-4 sm:grid-cols-2">
      <FormField
        id="monitor-interval"

        :model-value="form.intervalSeconds"
        @update:model-value="form.intervalSeconds = $event"
        label="Check interval (seconds)"
        type="number"
        :min="30"
        :max="86400"
        :disabled="disabled"
        :error="intervalError()"
        required
      />
      <FormField
        id="monitor-timeout"

        :model-value="form.timeoutSeconds"
        @update:model-value="form.timeoutSeconds = $event"
        label="Timeout (seconds)"
        type="number"
        :min="1"
        :max="Math.min(30, Number(form.intervalSeconds) || 30)"
        :disabled="disabled"
        :error="timeoutError()"
        required
      />
    </div>

    <FormField
      v-if="form.type === 'http'"
      id="monitor-expected-status"

      :model-value="form.expectedStatus"
      @update:model-value="form.expectedStatus = $event"
      label="Expected HTTP status (0 accepts any success status)"
      type="number"
      :min="0"
      :max="599"
      :disabled="disabled"
      :error="fieldErrors?.expectedStatus"
      required
    />

    <FormField
      v-if="form.type === 'tls'"
      id="monitor-tls-warn-days"

      :model-value="form.tlsWarnDays"
      @update:model-value="form.tlsWarnDays = $event"
      label="Warn when certificate expires within (days)"
      type="number"
      :min="1"
      :max="90"
      :disabled="disabled"
      :error="fieldErrors?.tlsWarnDays"
      required
    />

    <label class="flex items-center gap-2 text-sm">
      <input v-model="form.public" type="checkbox" class="size-4 accent-vigil-brand focus-visible:outline-2 focus-visible:outline-offset-2" :disabled="disabled" />
      Show this monitor on my public status page
    </label>

    <p v-if="validationError" class="text-sm text-vigil-down" role="alert">
      {{ validationError }}
    </p>

    <div v-if="fieldErrors?.public" class="text-sm text-vigil-down" role="alert">
      {{ fieldErrors.public }}
    </div>

    <button
      class="min-h-10 justify-self-start rounded-md bg-vigil-brand px-4 py-2 font-semibold text-white transition-colors hover:bg-vigil-brand-hover focus-visible:outline-2 focus-visible:outline-offset-2 disabled:cursor-not-allowed disabled:opacity-60"

      type="submit"
      :disabled="disabled || timeoutExceedsInterval"
    >
      {{ submitLabel }}
    </button>
  </form>
</template>
