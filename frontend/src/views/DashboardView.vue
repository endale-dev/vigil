<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { ApiError } from '../api/client'
import {
  createMonitor,
  deleteMonitor,
  listMonitors,
  pauseMonitor,
  resumeMonitor,
} from '../api/monitors'
import type { MonitorListDTO, MonitorRequest } from '../api/types'
import AppShell from '../components/AppShell.vue'
import ConfirmDialog from '../components/ConfirmDialog.vue'
import MonitorForm from '../components/MonitorForm.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { usePolling } from '../composables/usePolling'

const {
  data: monitorData,
  error: monitorError,
  isLoading,
  refresh,
} = usePolling(listMonitors, 15_000)

const monitors = computed(() => monitorData.value ?? [])
const loadError = computed(() => errorMessage(monitorError.value, 'Could not load monitors.'))
const createDialog = ref<HTMLDialogElement | null>(null)
const creating = ref(false)
const createError = ref('')
const formErrors = ref<Record<string, string | undefined>>({})
const actionError = ref('')
const pendingActionID = ref<string | null>(null)
const deleteTarget = ref<MonitorListDTO | null>(null)

function errorMessage(error: unknown, fallback: string): string {
  return error instanceof Error ? error.message : fallback
}

function openCreateDialog(): void {
  createError.value = ''
  formErrors.value = {}
  createDialog.value?.showModal()
}

function closeCreateDialog(): void {
  createDialog.value?.close()
}

async function submitCreate(input: MonitorRequest): Promise<void> {
  creating.value = true
  createError.value = ''
  formErrors.value = {}
  try {
    await createMonitor(input)
    closeCreateDialog()
    await refresh()
  } catch (error) {
    if (error instanceof ApiError && error.field) {
      formErrors.value = { [error.field]: error.message }
    } else {
      createError.value = errorMessage(error, 'Could not create this monitor.')
    }
  } finally {
    creating.value = false
  }
}

async function toggleMonitor(monitor: MonitorListDTO): Promise<void> {
  pendingActionID.value = monitor.id
  actionError.value = ''
  try {
    if (monitor.enabled) {
      await pauseMonitor(monitor.id)
    } else {
      await resumeMonitor(monitor.id)
    }
    await refresh()
  } catch (error) {
    actionError.value = errorMessage(error, 'Could not update this monitor.')
  } finally {
    pendingActionID.value = null
  }
}

async function confirmDelete(): Promise<void> {
  const monitor = deleteTarget.value
  if (!monitor) {
    return
  }
  pendingActionID.value = monitor.id
  actionError.value = ''
  try {
    await deleteMonitor(monitor.id)
    deleteTarget.value = null
    await refresh()
  } catch (error) {
    actionError.value = errorMessage(error, 'Could not delete this monitor.')
    deleteTarget.value = null
  } finally {
    pendingActionID.value = null
  }
}

function uptimeLabel(ratio: number | null): string {
  return ratio === null ? 'No data' : `${(ratio * 100).toFixed(2)}%`
}

function lastCheckedLabel(value: string | null): string {
  if (!value) {
    return 'No checks yet'
  }
  const checkedAt = new Date(value)
  return Number.isNaN(checkedAt.getTime()) ? 'Last check time unavailable' : checkedAt.toLocaleString()
}

function typeLabel(type: MonitorListDTO['type']): string {
  return type === 'tls' ? 'TLS certificate' : type.toUpperCase()
}
</script>

<template>
  <AppShell>
    <section class="mt-8" aria-label="Monitors" :aria-busy="isLoading">
      <div class="flex flex-wrap items-end justify-between gap-4">
        <div>
          <p class="text-sm text-slate-400">Your monitored services</p>
          <p class="mt-1 text-sm text-slate-400">Checks run on the interval you choose.</p>
        </div>
        <button
          class="min-h-10 rounded-md bg-emerald-700 px-4 py-2 font-medium text-white hover:bg-emerald-600 focus-visible:outline-2 focus-visible:outline-offset-2"
          type="button"
          @click="openCreateDialog"
        >
          Add monitor
        </button>
      </div>

      <p v-if="actionError" class="mt-5 rounded-md border border-red-800 bg-red-950/60 p-3 text-sm text-red-200" role="alert">
        {{ actionError }}
      </p>

      <div v-if="isLoading && !monitorData" class="mt-8 text-sm text-slate-300" role="status">
        Loading monitors…
      </div>
      <div v-else-if="monitorError && !monitorData" class="mt-8 rounded-lg border border-slate-800 bg-slate-900 p-6">
        <p class="text-sm text-red-200" role="alert">{{ loadError }}</p>
        <button class="mt-4 text-sm text-emerald-300 underline-offset-4 hover:underline focus-visible:outline-2" type="button" @click="refresh">
          Try again
        </button>
      </div>
      <div v-else-if="monitors.length === 0" class="mt-8 rounded-lg border border-dashed border-slate-700 bg-slate-900/50 px-6 py-12 text-center">
        <h2 class="text-lg font-semibold">No monitors yet</h2>
        <p class="mt-2 text-sm text-slate-300">Add a website or service to start checking its uptime.</p>
        <button
          class="mt-5 min-h-10 rounded-md bg-emerald-700 px-4 py-2 font-medium text-white hover:bg-emerald-600 focus-visible:outline-2 focus-visible:outline-offset-2"
          type="button"
          @click="openCreateDialog"
        >
          Add your first monitor
        </button>
      </div>
      <ul v-else class="mt-6 grid gap-4" aria-label="Your monitors">
        <li v-for="monitor in monitors" :key="monitor.id" class="rounded-lg border border-slate-800 bg-slate-900 p-5">
          <div class="flex flex-wrap items-start justify-between gap-4">
            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-3">
                <h2 class="text-lg font-semibold">{{ monitor.name }}</h2>
                <StatusBadge :status="monitor.status" />
                <span v-if="!monitor.enabled" class="rounded border border-slate-600 px-2 py-1 text-xs text-slate-300">Paused</span>
              </div>
              <p class="mt-2 break-all text-sm text-slate-300">{{ monitor.target }}</p>
            </div>
            <RouterLink
              class="text-sm text-emerald-300 underline-offset-4 hover:underline focus-visible:outline-2 focus-visible:outline-offset-4"
              :to="{ name: 'monitor-detail', params: { id: monitor.id } }"
            >
              View details
            </RouterLink>
          </div>

          <dl class="mt-5 grid gap-4 text-sm sm:grid-cols-3">
            <div>
              <dt class="text-slate-400">Type</dt>
              <dd class="mt-1">{{ typeLabel(monitor.type) }}</dd>
            </div>
            <div>
              <dt class="text-slate-400">Check interval</dt>
              <dd class="mt-1">Every {{ monitor.intervalSeconds }} seconds</dd>
            </div>
            <div>
              <dt class="text-slate-400">24-hour uptime</dt>
              <dd class="mt-1">{{ uptimeLabel(monitor.uptime24Hours.ratio) }}</dd>
            </div>
            <div class="sm:col-span-3">
              <dt class="text-slate-400">Last check</dt>
              <dd class="mt-1">{{ lastCheckedLabel(monitor.lastCheckedAt) }}</dd>
            </div>
          </dl>

          <div class="mt-5 flex flex-wrap gap-3 border-t border-slate-800 pt-4">
            <button
              class="min-h-9 rounded-md border border-slate-600 px-3 py-1.5 text-sm hover:bg-slate-800 focus-visible:outline-2 focus-visible:outline-offset-2 disabled:opacity-60"
              type="button"
              :disabled="pendingActionID !== null"
              @click="toggleMonitor(monitor)"
            >
              {{ pendingActionID === monitor.id ? 'Saving…' : monitor.enabled ? 'Pause' : 'Resume' }}
            </button>
            <button
              class="min-h-9 rounded-md border border-red-800 px-3 py-1.5 text-sm text-red-200 hover:bg-red-950 focus-visible:outline-2 focus-visible:outline-offset-2 disabled:opacity-60"
              type="button"
              :disabled="pendingActionID !== null"
              @click="deleteTarget = monitor"
            >
              Delete
            </button>
          </div>
        </li>
      </ul>
    </section>

    <dialog
      ref="createDialog"
      class="max-h-[90vh] w-[min(42rem,calc(100%-2rem))] overflow-y-auto rounded-lg border border-slate-700 bg-slate-900 p-6 text-slate-100 backdrop:bg-black/60"
      aria-labelledby="create-monitor-title"
    >
      <div class="mb-6 flex items-center justify-between gap-4">
        <h2 id="create-monitor-title" class="text-xl font-semibold">Add a monitor</h2>
        <button
          class="min-h-9 rounded-md border border-slate-600 px-3 text-sm hover:bg-slate-800 focus-visible:outline-2 focus-visible:outline-offset-2"
          type="button"
          aria-label="Close monitor form"
          @click="closeCreateDialog"
        >
          Close
        </button>
      </div>
      <p v-if="createError" class="mb-5 rounded-md border border-red-800 bg-red-950/60 p-3 text-sm text-red-200" role="alert">
        {{ createError }}
      </p>
      <MonitorForm
        :disabled="creating"
        :field-errors="formErrors"
        submit-label="Create monitor"
        @submit="submitCreate"
      />
    </dialog>

    <ConfirmDialog
      :open="deleteTarget !== null"
      title="Delete monitor?"
      :message="`Delete ${deleteTarget?.name ?? 'this monitor'} and its check history? This cannot be undone.`"
      confirm-label="Delete monitor"
      :busy="pendingActionID !== null"
      @cancel="deleteTarget = null"
      @confirm="confirmDelete"
    />
  </AppShell>
</template>
