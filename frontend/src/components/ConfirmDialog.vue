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
    class="w-[min(28rem,calc(100%-2rem))] rounded-lg border border-vigil-rule bg-vigil-surface p-6 text-vigil-ink shadow-xl backdrop:bg-vigil-ink/35"
    :aria-labelledby="'confirm-dialog-title'"
    :aria-describedby="'confirm-dialog-message'"
    @cancel.prevent="cancel"
  >
    <div class="grid gap-4">
      <h2 id="confirm-dialog-title" class="text-lg font-semibold">{{ title }}</h2>
      <p id="confirm-dialog-message" class="text-sm text-vigil-muted">{{ message }}</p>
      <div class="flex justify-end gap-3">
        <button
          class="min-h-10 rounded-md border border-vigil-rule px-4 py-2 hover:bg-vigil-paper focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-vigil-brand disabled:opacity-60"
          type="button"
          :disabled="busy"
          @click="cancel"
        >
          {{ cancelLabel }}
        </button>
        <button
          class="min-h-10 rounded-md bg-vigil-down px-4 py-2 font-semibold text-white hover:bg-vigil-down/90 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-vigil-down disabled:cursor-not-allowed disabled:opacity-60"
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
