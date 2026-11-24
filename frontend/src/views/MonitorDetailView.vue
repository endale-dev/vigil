<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { ApiError } from '../api/client'
import {
  deleteMonitor,
  getMonitor,
  listMonitorChecks,
  listMonitorIncidents,
  pauseMonitor,
  resumeMonitor,
  updateMonitor,
} from '../api/monitors'
import type { CheckDTO, IncidentDTO, MonitorDetailDTO, MonitorRequest, UptimeSummaryDTO } from '../api/types'
import AppShell from '../components/AppShell.vue'
import ConfirmDialog from '../components/ConfirmDialog.vue'
import MonitorForm from '../components/MonitorForm.vue'
import StatusBadge from '../components/StatusBadge.vue'
import UptimeStrip from '../components/UptimeStrip.vue'

const route = useRoute()
const router = useRouter()
const monitorID = computed(() => typeof route.params.id === 'string' ? route.params.id : '')
const monitor = ref<MonitorDetailDTO | null>(null)
const checks = ref<CheckDTO[]>([])
const incidents = ref<IncidentDTO[]>([])
const loading = ref(true)
const pageError = ref('')
const actionError = ref('')
const editError = ref('')
const fieldErrors = ref<Record<string, string | undefined>>({})
const editDialog = ref<HTMLDialogElement | null>(null)
const deleteOpen = ref(false)
const pending = ref(false)

async function loadDetail(): Promise<void> {
  const id = monitorID.value
  if (!id) {
    pageError.value = 'Monitor not found.'
    loading.value = false
    return
  }

  loading.value = true
  pageError.value = ''
  try {
    const [details, recentChecks, monitorIncidents] = await Promise.all([
      getMonitor(id),
      listMonitorChecks(id, 20),
      listMonitorIncidents(id, 20),
    ])
    monitor.value = details
    checks.value = recentChecks
    incidents.value = monitorIncidents
  } catch (error) {
    pageError.value = error instanceof ApiError && error.status === 404
      ? 'Monitor not found.'
      : error instanceof Error ? error.message : 'Could not load this monitor.'
  } finally {
    loading.value = false
  }
}

watch(monitorID, () => void loadDetail(), { immediate: true })

function uptimeLabel(summary: UptimeSummaryDTO): string {
  return summary.ratio === null ? 'No data' : `${(summary.ratio * 100).toFixed(2)}%`
}

function formatDate(value: string | null): string {
  if (!value) {
    return 'Not available'
  }
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? 'Time unavailable' : date.toLocaleString()
}

function typeLabel(type: MonitorDetailDTO['type']): string {
  return type === 'tls' ? 'TLS certificate' : type.toUpperCase()
}

function openEdit(): void {
  editError.value = ''
  fieldErrors.value = {}
  editDialog.value?.showModal()
}

async function saveEdit(input: MonitorRequest): Promise<void> {
  if (!monitor.value) {
    return
  }
  pending.value = true
  editError.value = ''
  fieldErrors.value = {}
  try {
    await updateMonitor(monitor.value.id, input)
    editDialog.value?.close()
    await loadDetail()
  } catch (error) {
    if (error instanceof ApiError && error.field) {
      fieldErrors.value = { [error.field]: error.message }
    } else {
      editError.value = error instanceof Error ? error.message : 'Could not save this monitor.'
    }
  } finally {
    pending.value = false
  }
}

async function toggleMonitor(): Promise<void> {
  if (!monitor.value) {
    return
  }
  pending.value = true
  actionError.value = ''
  try {
    if (monitor.value.enabled) {
      await pauseMonitor(monitor.value.id)
    } else {
      await resumeMonitor(monitor.value.id)
    }
    await loadDetail()
  } catch (error) {
    actionError.value = error instanceof Error ? error.message : 'Could not update this monitor.'
  } finally {
    pending.value = false
  }
}

async function confirmDelete(): Promise<void> {
  if (!monitor.value) {
    return
  }
  pending.value = true
  actionError.value = ''
  try {
    await deleteMonitor(monitor.value.id)
    deleteOpen.value = false
    await router.replace({ name: 'dashboard' })
  } catch (error) {
    actionError.value = error instanceof Error ? error.message : 'Could not delete this monitor.'
    deleteOpen.value = false
  } finally {
    pending.value = false
  }
}
</script>

<template>
  <AppShell>
    <div v-if="loading" class="mt-8 text-sm text-slate-300" role="status">Loading monitor…</div>
    <section v-else-if="pageError" class="mt-8 rounded-lg border border-slate-800 bg-slate-900 p-6">
      <p class="text-sm text-red-200" role="alert">{{ pageError }}</p>
      <RouterLink class="mt-4 inline-block text-sm text-emerald-300 underline-offset-4 hover:underline focus-visible:outline-2" :to="{ name: 'dashboard' }">
        Back to dashboard
      </RouterLink>
    </section>

    <section v-else-if="monitor" class="mt-8">
      <p v-if="actionError" class="mb-5 rounded-md border border-red-800 bg-red-950/60 p-3 text-sm text-red-200" role="alert">
        {{ actionError }}
      </p>

      <div class="flex flex-wrap items-start justify-between gap-4">
        <div class="min-w-0">
          <div class="flex flex-wrap items-center gap-3">
            <h2 class="text-2xl font-semibold">{{ monitor.name }}</h2>
            <StatusBadge :status="monitor.status" />
            <span v-if="!monitor.enabled" class="rounded border border-slate-600 px-2 py-1 text-xs text-slate-300">Paused</span>
          </div>
          <p class="mt-2 break-all text-sm text-slate-300">{{ monitor.target }}</p>
        </div>
        <div class="flex flex-wrap gap-2">
          <button
            class="min-h-10 rounded-md border border-slate-600 px-3 py-2 text-sm hover:bg-slate-800 focus-visible:outline-2 focus-visible:outline-offset-2 disabled:opacity-60"
            type="button"
            :disabled="pending"
            @click="openEdit"
          >
            Edit
          </button>
          <button
            class="min-h-10 rounded-md border border-slate-600 px-3 py-2 text-sm hover:bg-slate-800 focus-visible:outline-2 focus-visible:outline-offset-2 disabled:opacity-60"
            type="button"
            :disabled="pending"
            @click="toggleMonitor"
          >
            {{ pending ? 'Saving…' : monitor.enabled ? 'Pause' : 'Resume' }}
          </button>
          <button
            class="min-h-10 rounded-md border border-red-800 px-3 py-2 text-sm text-red-200 hover:bg-red-950 focus-visible:outline-2 focus-visible:outline-offset-2 disabled:opacity-60"
            type="button"
            :disabled="pending"
            @click="deleteOpen = true"
          >
            Delete
          </button>
        </div>
      </div>

      <dl class="mt-6 grid gap-4 rounded-lg border border-slate-800 bg-slate-900 p-5 text-sm sm:grid-cols-3">
        <div>
          <dt class="text-slate-400">Type</dt>
          <dd class="mt-1">{{ typeLabel(monitor.type) }}</dd>
        </div>
        <div>
          <dt class="text-slate-400">Check interval</dt>
          <dd class="mt-1">Every {{ monitor.intervalSeconds }} seconds</dd>
        </div>
        <div>
          <dt class="text-slate-400">Last check</dt>
          <dd class="mt-1">{{ formatDate(monitor.lastCheckedAt) }}</dd>
        </div>
      </dl>

      <section class="mt-8" aria-labelledby="uptime-title">
        <h3 id="uptime-title" class="text-lg font-semibold">Uptime</h3>
        <dl class="mt-4 grid gap-4 sm:grid-cols-3">
          <div v-for="period in [
            { label: '24 hours', value: monitor.uptime.last24Hours },
            { label: '7 days', value: monitor.uptime.last7Days },
            { label: '30 days', value: monitor.uptime.last30Days },
          ]" :key="period.label" class="rounded-lg border border-slate-800 bg-slate-900 p-5">
            <dt class="text-sm text-slate-400">Last {{ period.label }}</dt>
            <dd class="mt-2 text-2xl font-semibold">{{ uptimeLabel(period.value) }}</dd>
            <dd class="mt-1 text-xs text-slate-400">{{ period.value.up }} of {{ period.value.total }} checks up</dd>
          </div>
        </dl>
      </section>

      <section class="mt-8" aria-labelledby="checks-title">
        <h3 id="checks-title" class="text-lg font-semibold">Recent checks</h3>
        <div class="mt-3 rounded-lg border border-slate-800 bg-slate-900 p-5">
          <UptimeStrip :checks="checks" />
          <ul v-if="checks.length" class="mt-4 divide-y divide-slate-800">
            <li v-for="check in checks" :key="check.id" class="grid gap-1 py-3 text-sm sm:grid-cols-[1fr_auto] sm:items-center">
              <div>
                <span class="font-medium">{{ check.up ? 'Up' : 'Down' }}</span>
                <span class="ml-2 text-slate-300">{{ check.latencyMs }} ms</span>
                <p v-if="check.detail" class="mt-1 text-xs text-slate-400">{{ check.detail }}</p>
              </div>
              <time class="text-xs text-slate-400" :datetime="check.checkedAt">{{ formatDate(check.checkedAt) }}</time>
            </li>
          </ul>
        </div>
      </section>

      <section class="mt-8" aria-labelledby="incidents-title">
        <h3 id="incidents-title" class="text-lg font-semibold">Incidents</h3>
        <ul v-if="incidents.length" class="mt-3 divide-y divide-slate-800 rounded-lg border border-slate-800 bg-slate-900 px-5">
          <li v-for="incident in incidents" :key="incident.id" class="py-4">
            <div class="flex flex-wrap items-center justify-between gap-2">
              <p class="font-medium">{{ incident.cause }}</p>
              <span v-if="!incident.resolvedAt" class="rounded border border-red-800 px-2 py-1 text-xs text-red-200">Ongoing</span>
              <span v-else class="text-xs text-slate-400">Resolved {{ formatDate(incident.resolvedAt) }}</span>
            </div>
            <p class="mt-1 text-sm text-slate-400">Started {{ formatDate(incident.startedAt) }}</p>
          </li>
        </ul>
        <p v-else class="mt-3 rounded-lg border border-slate-800 bg-slate-900 p-5 text-sm text-slate-300">No incidents recorded.</p>
      </section>
    </section>

    <dialog
      ref="editDialog"
      class="max-h-[90vh] w-[min(42rem,calc(100%-2rem))] overflow-y-auto rounded-lg border border-slate-700 bg-slate-900 p-6 text-slate-100 backdrop:bg-black/60"
      aria-labelledby="edit-monitor-title"
    >
      <div class="mb-6 flex items-center justify-between gap-4">
        <h2 id="edit-monitor-title" class="text-xl font-semibold">Edit monitor</h2>
        <button
          class="min-h-9 rounded-md border border-slate-600 px-3 text-sm hover:bg-slate-800 focus-visible:outline-2 focus-visible:outline-offset-2"
          type="button"
          aria-label="Close monitor editor"
          @click="editDialog?.close()"
        >
          Close
        </button>
      </div>
      <p v-if="editError" class="mb-5 rounded-md border border-red-800 bg-red-950/60 p-3 text-sm text-red-200" role="alert">
        {{ editError }}
      </p>
      <MonitorForm
        v-if="monitor"
        :initial-value="monitor"
        :disabled="pending"
        :field-errors="fieldErrors"
        submit-label="Save changes"
        @submit="saveEdit"
      />
    </dialog>

    <ConfirmDialog
      :open="deleteOpen"
      title="Delete monitor?"
      :message="`Delete ${monitor?.name ?? 'this monitor'} and its check history? This cannot be undone.`"
      confirm-label="Delete monitor"
      :busy="pending"
      @cancel="deleteOpen = false"
      @confirm="confirmDelete"
    />
  </AppShell>
</template>
