<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'

const props = withDefaults(defineProps<{
  open: boolean
  title: string
  message: string
  confirmLabel?: string
  cancelLabel?: string
  busy?: boolean
  tone?: 'dark' | 'light'
}>(), {
  confirmLabel: 'Confirm',
  cancelLabel: 'Cancel',
  busy: false,
  tone: 'dark',
})

const emit = defineEmits<{
  cancel: []
  confirm: []
}>()

const dialog = ref<HTMLDialogElement | null>(null)

watch(() => props.open, async (open) => {
  await nextTick()
  if (!dialog.value) {
    return
  }
  if (open && !dialog.value.open) {
    dialog.value.showModal()
  } else if (!open && dialog.value.open) {
    dialog.value.close()
  }
}, { immediate: true, flush: 'post' })

function cancel(): void {
  emit('cancel')
}
</script>

<template>
  <dialog
    ref="dialog"
    class="w-[min(28rem,calc(100%-2rem))] rounded-2xl border p-6 shadow-2xl backdrop:bg-slate-950/35"
    :class="tone === 'light' ? 'border-vigil-line bg-white text-vigil-ink' : 'border-slate-700 bg-slate-900 text-slate-100'"
    :aria-labelledby="'confirm-dialog-title'"
    :aria-describedby="'confirm-dialog-message'"
    @cancel.prevent="cancel"
  >
    <div class="grid gap-4">
      <h2 id="confirm-dialog-title" class="text-lg font-semibold">{{ title }}</h2>
      <p id="confirm-dialog-message" class="text-sm" :class="tone === 'light' ? 'text-vigil-muted' : 'text-slate-300'">{{ message }}</p>
      <div class="flex justify-end gap-3">
        <button
          class="min-h-10 rounded-lg border px-4 py-2 focus-visible:outline-2 focus-visible:outline-offset-2 disabled:opacity-60"
          :class="tone === 'light' ? 'border-vigil-line hover:bg-vigil-canvas focus-visible:outline-vigil-brand' : 'border-slate-600'"
          type="button"
          :disabled="busy"
          @click="cancel"
        >
          {{ cancelLabel }}
        </button>
        <button
          class="min-h-10 rounded-lg px-4 py-2 font-semibold text-white focus-visible:outline-2 focus-visible:outline-offset-2 disabled:cursor-not-allowed disabled:opacity-60"
          :class="tone === 'light' ? 'bg-red-700 hover:bg-red-800 focus-visible:outline-red-700' : 'bg-red-700'"
          type="button"
          :disabled="busy"
          @click="emit('confirm')"
        >
          {{ confirmLabel }}
        </button>
      </div>
    </div>
  </dialog>
</template>
