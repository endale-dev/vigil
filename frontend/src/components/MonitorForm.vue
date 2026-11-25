<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { MonitorDTO, MonitorRequest, MonitorType } from '../api/types'
import FormField from './FormField.vue'
import SelectField from './SelectField.vue'

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

function updateType(value: string): void {
  const type = value as MonitorType
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
  let target = form.value.target.trim()
  if (form.value.type === 'http') {
    if (!/^[a-z][a-z0-9+.-]*:\/\//i.test(target)) target = `https://${target.replace(/^\/\//, '')}`
    try {
      const url = new URL(target)
      if (!['http:', 'https:'].includes(url.protocol) || !url.hostname || /\s/.test(target) || url.username || url.password) throw new Error('Invalid URL')
      target = url.href
    } catch {
      validationError.value = 'Enter a valid website address, such as example.com/health. Only HTTP and HTTPS URLs are supported.'
      return
    }
  }
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
    target,
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
    <div class="grid items-start gap-4 sm:grid-cols-[1.5fr_1fr]">
      <FormField
        id="monitor-name"
        :model-value="form.name"
        @update:model-value="form.name = $event"
        label="Monitor name"
        placeholder="e.g. Production API"
        autocomplete="off"
        autofocus
        :max-length="80"
        :disabled="disabled"
        :error="fieldErrors?.name"
        required
      />

      <SelectField id="monitor-type" label="Monitor type" :model-value="form.type" :disabled="disabled" :error="fieldErrors?.type"
        :options="[{ value: 'http', label: 'HTTP / HTTPS', description: 'Website or API availability' }, { value: 'tcp', label: 'TCP', description: 'Host and port connectivity' }, { value: 'tls', label: 'TLS certificate', description: 'Certificate validity and expiry' }]"
        @update:model-value="updateType" />

    </div>

    <FormField
      id="monitor-target"

      :model-value="form.target"
      @update:model-value="form.target = $event"
      :label="form.type === 'http' ? 'URL' : 'Host and port'"
      type="text"
      :inputmode="form.type === 'http' ? 'url' : 'text'"
      :placeholder="form.type === 'http' ? 'example.com/health' : 'example.com:443'"
      :hint="form.type === 'http' ? 'HTTPS is added automatically. Include http:// if your endpoint requires HTTP.' : 'Enter a hostname and port, such as example.com:443.'"
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
        :error="timeoutExceedsInterval ? 'Timeout must be within the check interval.' : timeoutError()"
        required
      />
    </div>

    <FormField
      v-if="form.type === 'http'"
      id="monitor-expected-status"

      :model-value="form.expectedStatus"
      @update:model-value="form.expectedStatus = $event"
      label="Expected HTTP status"
      hint="Enter 0 to accept any successful HTTP response."
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

    <div>
      <label class="flex items-center gap-2 text-sm">
        <input v-model="form.public" type="checkbox" class="size-4 accent-vigil-brand focus-visible:outline-2 focus-visible:outline-offset-2"
          :disabled="disabled" aria-describedby="monitor-visibility-hint" />
        Share on my public page
      </label>
      <p id="monitor-visibility-hint" class="mt-1 pl-6 text-xs text-vigil-muted">Anyone with the link can see this monitor’s name, status, uptime, and recent checks. Alert delivery is set separately.</p>
    </div>

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
      {{ disabled ? 'Saving…' : submitLabel }}
    </button>
  </form>
</template>
