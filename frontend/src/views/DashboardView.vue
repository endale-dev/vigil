<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import {
  Activity,
  CircleCheck,
  CircleDashed,
  CircleHelp,
  Globe,
  Network,
  Pause,
  Play,
  Plus,
  ShieldCheck,
  Trash2,
  TriangleAlert,
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
    return { title: 'No monitors yet', detail: 'Add a service to begin watching its availability.', icon: CircleDashed, tone: 'neutral' }
  }
  if (activeMonitors.value.length === 0) {
    return { title: 'Monitoring is paused', detail: 'Resume a monitor to start receiving fresh checks.', icon: CircleHelp, tone: 'neutral' }
  }
  if (downCount.value > 0) {
    return { title: 'Some services need attention', detail: `${downCount.value} active ${downCount.value === 1 ? 'monitor is' : 'monitors are'} currently down.`, icon: TriangleAlert, tone: 'danger' }
  }
  if (unknownCount.value > 0) {
    return { title: 'Waiting for the first checks', detail: 'Newly added services will show here as results arrive.', icon: Activity, tone: 'neutral' }
  }
  return { title: 'All systems operational', detail: 'Every active monitor is reporting a healthy response.', icon: CircleCheck, tone: 'success' }
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
  return ratio === null ? 'No data' : `${(ratio * 100).toFixed(2)}%`
}

function lastCheckedLabel(value: string | null): string {
  if (!value) {
    return 'No checks yet'
  }
  const checkedAt = new Date(value)
  return Number.isNaN(checkedAt.getTime()) ? 'Last check time unavailable' : checkedAt.toLocaleString()
}

</script>

<template>
  <WorkspaceShell title="Overview" description="Monitor service health across your stack">
    <template #header-action>
      <button
        class="hidden min-h-10 items-center gap-2 rounded-lg bg-vigil-brand px-4 text-sm font-semibold text-white shadow-sm shadow-blue-900/10 transition-colors hover:bg-vigil-brand-hover focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-vigil-brand sm:inline-flex"
        type="button"
        @click="openCreateDialog"
      >
        <Plus :size="17" :stroke-width="2.2" aria-hidden="true" />
        Add monitor
      </button>
    </template>

    <section class="grid gap-7" aria-label="Monitor overview" :aria-busy="isLoading">
      <header class="flex flex-wrap items-end justify-between gap-4">
        <div>
          <p class="text-xs font-semibold uppercase tracking-[0.15em] text-vigil-brand">Monitoring</p>
          <h1 class="mt-2 text-[1.8rem] font-semibold tracking-[-0.035em] text-vigil-ink sm:text-3xl">Your services at a glance</h1>
          <p class="mt-2 max-w-2xl text-sm leading-6 text-vigil-muted">A clear view of current health, recent checks, and uptime across your monitors.</p>
        </div>
        <button
          class="inline-flex min-h-10 items-center gap-2 rounded-lg bg-vigil-brand px-4 text-sm font-semibold text-white shadow-sm shadow-blue-900/10 transition-colors hover:bg-vigil-brand-hover focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-vigil-brand sm:hidden"
          type="button"
          @click="openCreateDialog"
        >
          <Plus :size="17" :stroke-width="2.2" aria-hidden="true" />
          Add monitor
        </button>
      </header>

      <div
        class="flex flex-col gap-4 rounded-2xl border p-5 sm:flex-row sm:items-center sm:justify-between sm:px-6 sm:py-5"
        :class="overallStatus.tone === 'success'
          ? 'border-emerald-200 bg-emerald-50/80'
          : overallStatus.tone === 'danger'
            ? 'border-red-200 bg-red-50/80'
            : 'border-vigil-line bg-white'"
        role="status"
      >
        <div class="flex items-start gap-4">
          <span
            class="grid size-11 shrink-0 place-items-center rounded-xl"
            :class="overallStatus.tone === 'success' ? 'bg-emerald-100 text-emerald-800' : overallStatus.tone === 'danger' ? 'bg-red-100 text-red-800' : 'bg-vigil-brand-soft text-vigil-brand'"
          >
            <component :is="overallStatus.icon" :size="21" :stroke-width="1.9" aria-hidden="true" />
          </span>
          <div>
            <p class="text-xs font-semibold uppercase tracking-[0.12em] text-vigil-muted">System status</p>
            <h2 class="mt-1 text-lg font-semibold tracking-tight text-vigil-ink">{{ overallStatus.title }}</h2>
            <p class="mt-1 text-sm text-vigil-muted">{{ overallStatus.detail }}</p>
          </div>
        </div>
        <p class="shrink-0 text-sm font-medium text-vigil-muted">
          {{ activeMonitors.length }} active <span class="mx-1 text-slate-300">·</span> {{ pausedCount }} paused
        </p>
      </div>

      <section class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3" aria-label="Key metrics">
        <article class="rounded-2xl border border-vigil-line bg-vigil-surface p-5 shadow-sm shadow-slate-900/[0.025]">
          <div class="flex items-center justify-between gap-3">
            <p class="text-sm font-medium text-vigil-muted">Active monitors</p>
            <span class="grid size-9 place-items-center rounded-lg bg-vigil-brand-soft text-vigil-brand">
              <Globe :size="17" :stroke-width="1.8" aria-hidden="true" />
            </span>
          </div>
          <p class="mt-5 text-3xl font-semibold tracking-[-0.04em] tabular-nums">{{ activeMonitors.length }}</p>
          <p class="mt-1 text-xs text-vigil-muted">{{ monitors.length }} total configured</p>
        </article>

        <article class="rounded-2xl border border-vigil-line bg-vigil-surface p-5 shadow-sm shadow-slate-900/[0.025]">
          <div class="flex items-center justify-between gap-3">
            <p class="text-sm font-medium text-vigil-muted">Services up</p>
            <span class="grid size-9 place-items-center rounded-lg bg-emerald-50 text-emerald-700">
              <CircleCheck :size="18" :stroke-width="1.9" aria-hidden="true" />
            </span>
          </div>
          <p class="mt-5 text-3xl font-semibold tracking-[-0.04em] tabular-nums">{{ upCount }}<span class="ml-1 text-base font-medium text-vigil-muted">/ {{ activeMonitors.length }}</span></p>
          <p class="mt-1 text-xs text-vigil-muted">{{ downCount }} down · {{ unknownCount }} awaiting checks</p>
        </article>

        <article class="rounded-2xl border border-vigil-line bg-vigil-surface p-5 shadow-sm shadow-slate-900/[0.025] sm:col-span-2 xl:col-span-1">
          <div class="flex items-center justify-between gap-3">
            <p class="text-sm font-medium text-vigil-muted">24-hour success rate</p>
            <span class="grid size-9 place-items-center rounded-lg bg-[#f0f3f8] text-vigil-sidebar">
              <Activity :size="18" :stroke-width="1.8" aria-hidden="true" />
            </span>
          </div>
          <p class="mt-5 text-3xl font-semibold tracking-[-0.04em] tabular-nums">{{ uptime24Hours === null ? '—' : `${(uptime24Hours * 100).toFixed(2)}%` }}</p>
          <p class="mt-1 text-xs text-vigil-muted">Weighted across available checks</p>
        </article>
      </section>

      <section class="overflow-hidden rounded-2xl border border-vigil-line bg-white shadow-sm shadow-slate-900/[0.025]" aria-labelledby="monitor-list-title">
        <div class="flex flex-wrap items-center justify-between gap-3 border-b border-vigil-line px-5 py-4 sm:px-6">
          <div>
            <h2 id="monitor-list-title" class="font-semibold tracking-tight">Monitors</h2>
            <p class="mt-1 text-xs text-vigil-muted">Updates automatically every 15 seconds</p>
          </div>
          <span class="rounded-full bg-vigil-canvas px-2.5 py-1 text-xs font-semibold tabular-nums text-vigil-muted">{{ monitors.length }}</span>
        </div>

        <p v-if="actionError" class="m-5 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800 sm:mx-6" role="alert">
          {{ actionError }}
        </p>

        <div v-if="isLoading && !monitorData" class="px-6 py-12 text-center text-sm text-vigil-muted" role="status">
          Loading monitors…
        </div>
        <div v-else-if="monitorError && !monitorData" class="px-6 py-10 text-center">
          <p class="text-sm text-red-700" role="alert">{{ loadError }}</p>
          <button class="mt-3 text-sm font-semibold text-vigil-brand hover:underline focus-visible:outline-2 focus-visible:outline-offset-2" type="button" @click="refresh">
            Try again
          </button>
        </div>
        <div v-else-if="monitors.length === 0" class="px-6 py-14 text-center sm:py-16">
          <span class="mx-auto grid size-12 place-items-center rounded-2xl bg-vigil-brand-soft text-vigil-brand">
            <Activity :size="22" :stroke-width="1.8" aria-hidden="true" />
          </span>
          <h3 class="mt-4 text-base font-semibold">Start with your first monitor</h3>
          <p class="mx-auto mt-2 max-w-sm text-sm leading-6 text-vigil-muted">Add a website, service, or certificate and Vigil will keep an eye on it.</p>
          <button class="mt-5 inline-flex min-h-10 items-center gap-2 rounded-lg bg-vigil-brand px-4 text-sm font-semibold text-white transition-colors hover:bg-vigil-brand-hover focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-vigil-brand" type="button" @click="openCreateDialog">
            <Plus :size="17" aria-hidden="true" />
            Add your first monitor
          </button>
        </div>
        <ul v-else class="divide-y divide-vigil-line" aria-label="Your monitors">
          <li v-for="monitor in monitors" :key="monitor.id" class="grid items-center gap-x-5 gap-y-4 px-5 py-5 transition-colors hover:bg-slate-50/70 sm:px-6 lg:grid-cols-[minmax(0,1.6fr)_minmax(8rem,0.8fr)_minmax(7rem,0.7fr)_minmax(8rem,0.9fr)_auto]">
            <div class="flex min-w-0 items-center gap-3">
              <span class="grid size-10 shrink-0 place-items-center rounded-xl border border-vigil-line bg-vigil-canvas text-vigil-muted">
                <component :is="monitor.type === 'http' ? Globe : monitor.type === 'tcp' ? Network : ShieldCheck" :size="18" :stroke-width="1.8" aria-hidden="true" />
              </span>
              <div class="min-w-0">
                <RouterLink class="block truncate text-sm font-semibold text-vigil-ink hover:text-vigil-brand focus-visible:rounded-sm focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-vigil-brand" :to="{ name: 'monitor-detail', params: { id: monitor.id } }">
                  {{ monitor.name }}
                </RouterLink>
                <p class="mt-1 truncate text-xs text-vigil-muted" :title="monitor.target">{{ monitor.target }}</p>
              </div>
            </div>

            <div class="flex items-center gap-2 lg:block">
              <p class="mb-1 hidden text-[11px] font-medium uppercase tracking-wide text-vigil-muted lg:block">Status</p>
              <StatusBadge :status="monitor.status" tone="light" />
              <span v-if="!monitor.enabled" class="rounded-full border border-slate-200 bg-slate-100 px-2.5 py-1 text-xs font-medium text-slate-600">Paused</span>
            </div>

            <div>
              <p class="mb-1 text-[11px] font-medium uppercase tracking-wide text-vigil-muted">24h uptime</p>
              <p class="text-sm font-semibold tabular-nums text-vigil-ink">{{ uptimeLabel(monitor.uptime24Hours.ratio) }}</p>
            </div>

            <div class="hidden lg:block">
              <p class="mb-1 text-[11px] font-medium uppercase tracking-wide text-vigil-muted">Last check</p>
              <p class="text-sm text-vigil-ink">{{ lastCheckedLabel(monitor.lastCheckedAt) }}</p>
              <p class="mt-1 text-xs text-vigil-muted">Every {{ monitor.intervalSeconds }} seconds</p>
            </div>

            <div class="flex items-center justify-end gap-1">
              <button
                class="grid size-9 place-items-center rounded-lg text-vigil-muted transition-colors hover:bg-vigil-brand-soft hover:text-vigil-brand focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-vigil-brand disabled:opacity-50"
                type="button"
                :aria-label="monitor.enabled ? `Pause ${monitor.name}` : `Resume ${monitor.name}`"
                :title="monitor.enabled ? 'Pause monitor' : 'Resume monitor'"
                :disabled="pendingActionID !== null"
                @click="toggleMonitor(monitor)"
              >
                <component :is="monitor.enabled ? Pause : Play" :size="17" :stroke-width="1.9" aria-hidden="true" />
              </button>
              <button
                class="grid size-9 place-items-center rounded-lg text-vigil-muted transition-colors hover:bg-red-50 hover:text-red-700 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-red-700 disabled:opacity-50"
                type="button"
                :aria-label="`Delete ${monitor.name}`"
                title="Delete monitor"
                :disabled="pendingActionID !== null"
                @click="deleteTarget = monitor"
              >
                <Trash2 :size="17" :stroke-width="1.8" aria-hidden="true" />
              </button>
            </div>
          </li>
        </ul>
      </section>
    </section>

    <dialog
      ref="createDialog"
      class="max-h-[90vh] w-[min(42rem,calc(100%-2rem))] overflow-y-auto rounded-2xl border border-vigil-line bg-white p-6 text-vigil-ink shadow-2xl backdrop:bg-slate-950/35 sm:p-7"
      aria-labelledby="create-monitor-title"
    >
      <div class="mb-6 flex items-center justify-between gap-4">
        <div>
          <p class="text-xs font-semibold uppercase tracking-[0.13em] text-vigil-brand">New service</p>
          <h2 id="create-monitor-title" class="mt-1 text-xl font-semibold tracking-tight">Add a monitor</h2>
        </div>
        <button
          class="min-h-9 rounded-lg border border-vigil-line px-3 text-sm text-vigil-muted transition-colors hover:bg-vigil-canvas hover:text-vigil-ink focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-vigil-brand"
          type="button"
          @click="closeCreateDialog"
        >
          Close
        </button>
      </div>
      <p v-if="createError" class="mb-5 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-800" role="alert">
        {{ createError }}
      </p>
      <MonitorForm
        tone="light"
        :disabled="creating"
        :field-errors="formErrors"
        submit-label="Create monitor"
        @submit="submitCreate"
      />
    </dialog>

    <ConfirmDialog
      tone="light"
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
