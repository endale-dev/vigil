<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import {
  Activity,
  Pause,
  Play,
  Plus,
  Trash2,
} from '@lucide/vue'
import { ApiError } from '../api/client'
import {
  createMonitor,
  deleteMonitor,
  listMonitors,
  pauseMonitor,
  resumeMonitor,
} from '../api/monitors'
import type { MonitorListDTO, MonitorRequest } from '../api/types'
import ConfirmDialog from '../components/ConfirmDialog.vue'
import MonitorForm from '../components/MonitorForm.vue'
import StatusBadge from '../components/StatusBadge.vue'
import WorkspaceShell from '../components/WorkspaceShell.vue'
import { usePolling } from '../composables/usePolling'

const {
  data: monitorData,
  error: monitorError,
  isLoading,
  refresh,
} = usePolling(listMonitors, 15_000)

const monitors = computed(() => monitorData.value ?? [])
const loadError = computed(() => errorMessage(monitorError.value, 'Could not load monitors.'))
const activeMonitors = computed(() => monitors.value.filter((monitor) => monitor.enabled))
const pausedCount = computed(() => monitors.value.length - activeMonitors.value.length)
const upCount = computed(() => activeMonitors.value.filter((monitor) => monitor.status === 'up').length)
const downCount = computed(() => activeMonitors.value.filter((monitor) => monitor.status === 'down').length)
const unknownCount = computed(() => activeMonitors.value.filter((monitor) => monitor.status === 'unknown').length)
const uptime24Hours = computed(() => {
  const summaries = monitors.value.map((monitor) => monitor.uptime24Hours)
  const total = summaries.reduce((sum, summary) => sum + summary.total, 0)
  if (total === 0) {
    return null
  }
  const up = summaries.reduce((sum, summary) => sum + summary.up, 0)
  return up / total
})
const overallStatus = computed(() => {
  if (monitors.value.length === 0) {
    return { title: 'Monitoring is ready', detail: 'Add your first service to begin collecting checks.', tone: 'neutral' }
  }
  if (activeMonitors.value.length === 0) {
    return { title: 'Monitoring is paused', detail: 'Resume a monitor to start receiving fresh checks.', tone: 'neutral' }
  }
  if (downCount.value > 0) {
    return { title: 'Some services need attention', detail: `${downCount.value} active ${downCount.value === 1 ? 'monitor is' : 'monitors are'} currently down.`, tone: 'danger' }
  }
  if (unknownCount.value > 0) {
    return { title: 'Waiting for the first checks', detail: 'Newly added services will show here as results arrive.', tone: 'neutral' }
  }
  return { title: 'All systems operational', detail: 'Every active monitor is reporting a healthy response.', tone: 'success' }
})
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
  return ratio === null ? 'Awaiting checks' : `${(ratio * 100).toFixed(2)}%`
}

function lastCheckedLabel(value: string | null): string {
  if (!value) {
    return 'First check pending'
  }
  const checkedAt = new Date(value)
  return Number.isNaN(checkedAt.getTime())
    ? 'Time unavailable'
    : checkedAt.toLocaleString(undefined, { month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit' })
}

</script>

<template>
  <WorkspaceShell>
    <section class="grid gap-0" aria-label="Monitor overview" :aria-busy="isLoading">
      <header class="flex flex-wrap items-end justify-between gap-4 pb-7">
        <div>
          <p class="text-sm font-medium text-vigil-brand">Your workspace</p>
          <h1 class="mt-2 text-[1.9rem] font-semibold leading-tight tracking-[-0.04em] text-vigil-ink sm:text-3xl">Overview</h1>
        </div>
        <button
          class="inline-flex min-h-11 items-center gap-2 rounded-md bg-vigil-brand px-4 text-sm font-semibold text-white transition-colors hover:bg-vigil-brand-hover focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-vigil-brand"
          type="button"
          @click="openCreateDialog"
        >
          <Plus :size="17" :stroke-width="2" aria-hidden="true" />
          Add monitor
        </button>
      </header>

      <div
        class="flex flex-wrap items-center justify-between gap-x-6 gap-y-2 border-y py-4"
        :class="overallStatus.tone === 'danger' ? 'border-vigil-down/35 bg-vigil-down-soft/45' : 'border-vigil-rule'"
        role="status"
      >
        <div class="flex min-w-0 items-center gap-3 px-3 sm:px-0">
          <span
            class="size-2.5 shrink-0 rounded-full"
            :class="overallStatus.tone === 'success' ? 'bg-vigil-up' : overallStatus.tone === 'danger' ? 'bg-vigil-down' : 'bg-vigil-unknown'"
            aria-hidden="true"
          />
          <div class="flex min-w-0 flex-wrap items-baseline gap-x-2 gap-y-1">
            <p class="font-semibold" :class="overallStatus.tone === 'danger' ? 'text-vigil-down' : 'text-vigil-ink'">{{ overallStatus.title }}</p>
            <p v-if="overallStatus.detail" class="text-sm text-vigil-muted">{{ overallStatus.detail }}</p>
          </div>
        </div>
        <p v-if="monitors.length > 0" class="px-3 text-sm text-vigil-muted sm:px-0">
          {{ activeMonitors.length }} active <span class="mx-1 text-vigil-rule">·</span> {{ pausedCount }} paused
        </p>
      </div>

      <section v-if="monitors.length > 0" class="grid grid-cols-2 border-b border-vigil-rule sm:grid-cols-4" aria-label="Monitor summary">
        <div class="border-b border-vigil-rule py-4 pr-4 sm:border-b-0 sm:pr-6">
          <p class="text-sm text-vigil-muted">Active</p>
          <p class="mt-1 font-vigil-mono text-2xl font-medium tabular-nums text-vigil-ink">{{ activeMonitors.length }}</p>
        </div>
        <div class="border-b border-l border-vigil-rule py-4 pl-4 sm:border-b-0 sm:px-6">
          <p class="text-sm text-vigil-muted">Up</p>
          <p class="mt-1 font-vigil-mono text-2xl font-medium tabular-nums text-vigil-up">{{ upCount }}</p>
        </div>
        <div class="py-4 pr-4 sm:border-l sm:border-vigil-rule sm:px-6">
          <p class="text-sm text-vigil-muted">Down</p>
          <p class="mt-1 font-vigil-mono text-2xl font-medium tabular-nums" :class="downCount ? 'text-vigil-down' : 'text-vigil-ink'">{{ downCount }}</p>
        </div>
        <div class="border-l border-vigil-rule py-4 pl-4 sm:pl-6">
          <p class="text-sm text-vigil-muted">24h success</p>
          <p v-if="uptime24Hours !== null" class="mt-1 font-vigil-mono text-2xl font-medium tabular-nums text-vigil-ink">{{ `${(uptime24Hours * 100).toFixed(2)}%` }}</p>
          <p v-else class="mt-2 text-sm font-medium text-vigil-muted">Awaiting checks</p>
        </div>
      </section>

      <section class="pt-8" aria-labelledby="monitor-list-title">
        <div class="flex items-baseline justify-between gap-4 pb-3">
          <h2 id="monitor-list-title" class="text-lg font-semibold tracking-tight">Monitors</h2>
          <p v-if="monitors.length" class="text-xs text-vigil-muted">Refreshes every 15 seconds</p>
        </div>

        <p v-if="actionError" class="mb-4 border-l-2 border-vigil-down bg-vigil-down-soft px-4 py-3 text-sm text-vigil-down" role="alert">
          {{ actionError }}
        </p>

        <div v-if="isLoading && !monitorData" class="border-y border-vigil-rule py-10 text-sm text-vigil-muted" role="status">
          Loading monitors…
        </div>
        <div v-else-if="monitorError && !monitorData" class="border-y border-vigil-rule py-8">
          <p class="text-sm text-vigil-down" role="alert">{{ loadError }}</p>
          <button class="mt-3 text-sm font-semibold text-vigil-brand underline-offset-4 hover:underline focus-visible:outline-2 focus-visible:outline-offset-2" type="button" @click="refresh">
            Try again
          </button>
        </div>
        <div v-else-if="monitors.length === 0" class="flex items-start gap-4 border-y border-vigil-rule py-7">
          <span class="grid size-10 shrink-0 place-items-center rounded-md bg-vigil-brand-soft text-vigil-brand" aria-hidden="true">
            <Activity :size="18" :stroke-width="1.8" />
          </span>
          <div>
            <h3 class="text-sm font-semibold text-vigil-ink">Your first monitor starts here</h3>
            <p class="mt-1 text-sm text-vigil-muted">Add a service to see its availability and check history.</p>
          </div>
        </div>
        <div v-else>
          <div class="hidden grid-cols-[minmax(0,1.7fr)_minmax(7.5rem,.7fr)_minmax(5.5rem,.6fr)_minmax(8rem,.8fr)_auto] gap-4 border-y border-vigil-rule py-2.5 text-xs font-medium text-vigil-muted xl:grid">
            <span>Service</span>
            <span>Status</span>
            <span>24h uptime</span>
            <span>Last check</span>
            <span class="sr-only">Actions</span>
          </div>
          <ul aria-label="Your monitors">
            <li v-for="monitor in monitors" :key="monitor.id" class="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-4 gap-y-3 border-b border-vigil-rule py-4 xl:grid-cols-[minmax(0,1.7fr)_minmax(7.5rem,.7fr)_minmax(5.5rem,.6fr)_minmax(8rem,.8fr)_auto] xl:gap-4">
              <div class="min-w-0">
                <RouterLink class="block truncate text-sm font-semibold text-vigil-ink underline-offset-4 hover:text-vigil-brand hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-vigil-brand" :to="{ name: 'monitor-detail', params: { id: monitor.id } }">
                  {{ monitor.name }}
                </RouterLink>
                <p class="mt-1 truncate text-xs text-vigil-muted" :title="monitor.target">{{ monitor.target }} <span class="mx-1 text-vigil-rule">·</span> {{ monitor.type === 'tls' ? 'TLS' : monitor.type.toUpperCase() }}</p>
              </div>

              <div class="justify-self-end xl:justify-self-start">
                <StatusBadge :status="monitor.status" />
                <span v-if="!monitor.enabled" class="ml-1 text-xs font-medium text-vigil-unknown">Paused</span>
              </div>

              <div class="col-span-1">
                <p class="text-[11px] text-vigil-muted xl:hidden">24h</p>
                <p class="font-vigil-mono text-sm tabular-nums text-vigil-ink">{{ uptimeLabel(monitor.uptime24Hours.ratio) }}</p>
              </div>

              <div class="hidden xl:block">
                <p class="text-sm text-vigil-ink">{{ lastCheckedLabel(monitor.lastCheckedAt) }}</p>
                <p class="mt-0.5 text-xs text-vigil-muted">Every {{ monitor.intervalSeconds }} sec</p>
              </div>

              <div class="col-span-2 flex items-center justify-end gap-3 xl:col-span-1">
                <button
                  class="inline-flex min-h-9 items-center gap-1.5 rounded-sm px-1 text-xs font-medium text-vigil-muted hover:text-vigil-brand focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-vigil-brand disabled:opacity-50"
                  type="button"
                  :aria-label="monitor.enabled ? `Pause ${monitor.name}` : `Resume ${monitor.name}`"
                  :disabled="pendingActionID !== null"
                  @click="toggleMonitor(monitor)"
                >
                  <component :is="monitor.enabled ? Pause : Play" :size="15" aria-hidden="true" />
                  {{ pendingActionID === monitor.id ? 'Saving…' : monitor.enabled ? 'Pause' : 'Resume' }}
                </button>
                <button
                  class="inline-flex min-h-9 items-center gap-1.5 rounded-sm px-1 text-xs font-medium text-vigil-muted hover:text-vigil-down focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-vigil-down disabled:opacity-50"
                  type="button"
                  :aria-label="`Delete ${monitor.name}`"
                  :disabled="pendingActionID !== null"
                  @click="deleteTarget = monitor"
                >
                  <Trash2 :size="15" aria-hidden="true" />
                  Delete
                </button>
              </div>
            </li>
          </ul>
        </div>
      </section>
    </section>

    <dialog ref="createDialog" class="max-h-[90vh] w-[min(42rem,calc(100%-2rem))] overflow-y-auto rounded-lg border border-vigil-rule bg-vigil-surface p-6 text-vigil-ink shadow-xl backdrop:bg-vigil-ink/35 sm:p-7" aria-labelledby="create-monitor-title">
      <div class="mb-6 flex items-center justify-between gap-4 border-b border-vigil-rule pb-4">
        <div>
          <p class="text-xs font-medium text-vigil-brand">New service</p>
          <h2 id="create-monitor-title" class="mt-1 text-xl font-semibold tracking-tight">Add a monitor</h2>
        </div>
        <button class="min-h-9 rounded-sm px-2 text-sm text-vigil-muted hover:text-vigil-ink focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-vigil-brand" type="button" @click="closeCreateDialog">
          Close
        </button>
      </div>
      <p v-if="createError" class="mb-5 border-l-2 border-vigil-down bg-vigil-down-soft px-4 py-3 text-sm text-vigil-down" role="alert">
        {{ createError }}
      </p>
      <MonitorForm :disabled="creating" :field-errors="formErrors" submit-label="Create monitor" @submit="submitCreate" />
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
  </WorkspaceShell>
</template>
