<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'

const props = withDefaults(defineProps<{
  open: boolean
  title: string
  message: string
  confirmLabel?: string
  cancelLabel?: string
  busy?: boolean
}>(), {
  confirmLabel: 'Confirm',
  cancelLabel: 'Cancel',
  busy: false,
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
    class="w-[min(28rem,calc(100%-2rem))] rounded-lg border border-slate-700 bg-slate-900 p-6 text-slate-100 backdrop:bg-black/60"
    :aria-labelledby="'confirm-dialog-title'"
    :aria-describedby="'confirm-dialog-message'"
    @cancel.prevent="cancel"
  >
    <div class="grid gap-4">
      <h2 id="confirm-dialog-title" class="text-lg font-semibold">{{ title }}</h2>
      <p id="confirm-dialog-message" class="text-sm text-slate-300">{{ message }}</p>
      <div class="flex justify-end gap-3">
        <button
          class="min-h-10 rounded-md border border-slate-600 px-4 py-2 focus-visible:outline-2 focus-visible:outline-offset-2 disabled:opacity-60"
          type="button"
          :disabled="busy"
          @click="cancel"
        >
          {{ cancelLabel }}
        </button>
        <button
          class="min-h-10 rounded-md bg-red-700 px-4 py-2 font-medium focus-visible:outline-2 focus-visible:outline-offset-2 disabled:cursor-not-allowed disabled:opacity-60"
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
