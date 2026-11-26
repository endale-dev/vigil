<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { Activity, ArrowLeft, ChartNoAxesCombined, Clock3, Pause, Pencil, Play, RefreshCw, Timer, Trash2, TriangleAlert, Waypoints, X } from '@lucide/vue'
import { checkIntervalLabel } from '../format'
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
import type { MonitorDetailDTO, MonitorRequest, UptimeSummaryDTO } from '../api/types'
import WorkspaceShell from '../components/WorkspaceShell.vue'
import ConfirmDialog from '../components/ConfirmDialog.vue'
import LoadingStatus from '../components/LoadingStatus.vue'
import MonitorForm from '../components/MonitorForm.vue'
import PaginatedList from '../components/PaginatedList.vue'
import StatusBadge from '../components/StatusBadge.vue'
import UptimeStrip from '../components/UptimeStrip.vue'
import { useAsyncResource } from '../composables/useAsyncResource'

const route = useRoute()
const router = useRouter()
const monitorID = route.params.id as string
const historyLimit = 20
const { data: monitor, loading, error: detailError, refresh: refreshMonitor } =
  useAsyncResource(signal => getMonitor(monitorID, signal))
const { data: checks, loading: loadingChecks, error: checksError, refresh: refreshChecks } =
  useAsyncResource(signal => listMonitorChecks(monitorID, historyLimit, signal))
const { data: incidents, loading: loadingIncidents, error: incidentsError, refresh: refreshIncidents } =
  useAsyncResource(signal => listMonitorIncidents(monitorID, historyLimit, signal))
const refreshing = computed(() => loading.value || loadingChecks.value || loadingIncidents.value)
const notFound = computed(() => detailError.value instanceof ApiError && detailError.value.status === 404)
const actionError = ref('')
const editError = ref('')
const fieldErrors = ref<Record<string, string | undefined>>({})
const editDialog = ref<HTMLDialogElement | null>(null)
const formVersion = ref(0)
const deleteOpen = ref(false)
const pending = ref<'edit' | 'toggle' | 'delete' | null>(null)

function refreshDetail() {
  return Promise.all([refreshMonitor(), refreshChecks(), refreshIncidents()])
}

function uptimeLabel(summary: UptimeSummaryDTO): string {
  return summary.ratio === null ? 'Awaiting checks' : `${(summary.ratio * 100).toFixed(2)}%`
}

function formatDate(value: string | null): string {
  if (!value) {
    return 'First check pending'
  }
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? 'Time unavailable' : date.toLocaleString()
}

function typeLabel(type: MonitorDetailDTO['type']): string {
  return type === 'tls' ? 'TLS certificate' : type.toUpperCase()
}

async function openEdit(): Promise<void> {
  formVersion.value++
  editError.value = ''
  fieldErrors.value = {}
  await nextTick()
  editDialog.value?.showModal()
}

async function saveEdit(input: MonitorRequest): Promise<void> {
  if (!monitor.value) {
    return
  }
  pending.value = 'edit'
  editError.value = ''
  fieldErrors.value = {}
  try {
    const updated = await updateMonitor(monitorID, input)
    monitor.value = { ...monitor.value, ...updated }
    editDialog.value?.close()
    if (route.params.id === monitorID) void refreshDetail()
  } catch (error) {
    if (error instanceof ApiError && error.field) {
      fieldErrors.value = { [error.field]: error.message }
    } else {
      editError.value = error instanceof Error ? error.message : 'Could not save this monitor.'
    }
  } finally {
    pending.value = null
  }
}

async function toggleMonitor(): Promise<void> {
  if (!monitor.value) {
    return
  }
  pending.value = 'toggle'
  actionError.value = ''
  try {
    const toggle = monitor.value.enabled ? pauseMonitor : resumeMonitor
    const updated = await toggle(monitorID)
    monitor.value = { ...monitor.value, ...updated }
    if (route.params.id === monitorID) void refreshDetail()
  } catch (error) {
    actionError.value = error instanceof Error ? error.message : 'Could not update this monitor.'
  } finally {
    pending.value = null
  }
}

async function confirmDelete(): Promise<void> {
  if (!monitor.value) {
    return
  }
  pending.value = 'delete'
  actionError.value = ''
  try {
    await deleteMonitor(monitor.value.id)
    deleteOpen.value = false
    if (route.params.id === monitorID) await router.replace({ name: 'dashboard' })
  } catch (error) {
    actionError.value = error instanceof Error ? error.message : 'Could not delete this monitor.'
    deleteOpen.value = false
  } finally {
    pending.value = null
  }
}
</script>

<template>
  <WorkspaceShell>
    <div class="detail-navigation">
      <RouterLink class="detail-back" :to="{ name: 'dashboard' }"><ArrowLeft :size="15" aria-hidden="true" /> All monitors</RouterLink>
      <button v-if="monitor && !notFound" type="button" class="detail-refresh" :disabled="refreshing || !!pending" @click="refreshDetail">
        <RefreshCw :size="14" :class="{ refreshing }" aria-hidden="true" /> {{ refreshing ? 'Refreshing…' : 'Refresh' }}
      </button>
    </div>
    <LoadingStatus v-if="!monitor && loading" class="console-panel p-5">Loading monitor…</LoadingStatus>
    <div v-if="detailError" class="console-notice danger" role="alert">
      {{ notFound ? 'Monitor not found.' : detailError.message }}
      <button v-if="!notFound" type="button" class="text-action" @click="refreshMonitor">Try again</button>
    </div>

    <section v-if="monitor && !notFound">
      <p v-if="actionError" class="console-notice danger" role="alert">{{ actionError }}</p>
      <header class="console-page-head detail-head">
        <div class="min-w-0">
          <div class="flex flex-wrap items-center gap-3"><h1 class="console-title">{{ monitor.name }}</h1>
            <StatusBadge plain :status="monitor.enabled ? monitor.status : 'paused'" />
          </div>
          <p class="console-description break-all">{{ monitor.target }}</p>
        </div>
        <div class="detail-actions">
          <button class="console-button primary" type="button" :disabled="!!pending" @click="openEdit"><Pencil :size="15" aria-hidden="true" /> Edit monitor</button>
          <button class="console-button secondary" type="button" :disabled="!!pending" @click="toggleMonitor">
            <component :is="monitor.enabled ? Pause : Play" :size="15" aria-hidden="true" />
            {{ pending === 'toggle' ? (monitor.enabled ? 'Pausing…' : 'Resuming…') : monitor.enabled ? 'Pause' : 'Resume' }}
          </button>
          <button class="icon-control destructive" type="button" aria-label="Delete monitor" title="Delete monitor" :disabled="!!pending" @click="deleteOpen = true"><Trash2 :size="16" aria-hidden="true" /></button>
        </div>
      </header>
      <dl class="detail-metadata console-panel">
        <div><dt><Waypoints :size="15" aria-hidden="true" />Monitor type</dt><dd>{{ typeLabel(monitor.type) }}</dd></div>
        <div><dt><Timer :size="15" aria-hidden="true" />Check interval</dt><dd>{{ checkIntervalLabel(monitor.intervalSeconds) }}</dd></div>
        <div><dt><Clock3 :size="15" aria-hidden="true" />Last check</dt><dd>{{ formatDate(monitor.lastCheckedAt) }}</dd></div>
      </dl>
      <section class="console-panel detail-uptime" aria-labelledby="uptime-title">
        <div class="panel-title"><ChartNoAxesCombined :size="18" aria-hidden="true" /><h2 id="uptime-title">Uptime</h2></div>
        <dl>
          <div v-for="period in [
            { label: 'Last 24 hours', value: monitor.uptime.last24Hours },
            { label: 'Last 7 days', value: monitor.uptime.last7Days },
            { label: 'Last 30 days', value: monitor.uptime.last30Days },
          ]" :key="period.label">
            <dt>{{ period.label }}</dt><dd>{{ period.value.total ? uptimeLabel(period.value) : '—' }}</dd>
            <dd class="uptime-caption">{{ period.value.total ? period.value.up + ' of ' + period.value.total + ' checks up' : 'Awaiting checks' }}</dd>
          </div>
        </dl>
      </section>
      <div class="detail-history">
        <section class="console-panel history-card" aria-labelledby="checks-title">
          <div class="panel-title"><Activity :size="18" aria-hidden="true" /><h2 id="checks-title">Recent checks</h2></div>
          <LoadingStatus v-if="loadingChecks && !checks">Loading checks…</LoadingStatus>
          <div v-if="checksError" class="console-notice danger" role="alert">
            {{ checksError.message }} <button type="button" class="text-action" @click="refreshChecks">Retry checks</button>
          </div>
          <template v-if="checks">
            <UptimeStrip v-if="checks.length || monitor.enabled" :checks="checks" />
            <p v-else class="history-empty">Resume this monitor to collect checks.</p>
          </template>
          <PaginatedList v-if="checks?.length" v-slot="{ items }" :items="checks" :page-size="5" label="recent checks">
            <ul class="check-list">
              <li v-for="check in items" :key="check.id">
                <div><span class="check-state" :class="{ failed: !check.up }">{{ check.up ? 'Up' : 'Down' }}</span> <span class="number">{{ check.latencyMs }} ms</span>
                  <p v-if="check.detail">{{ check.detail }}</p>
                </div><time :datetime="check.checkedAt">{{ formatDate(check.checkedAt) }}</time>
              </li>
            </ul>
          </PaginatedList>
        </section>
        <section class="console-panel history-card" aria-labelledby="incidents-title">
          <div class="panel-title"><TriangleAlert :size="18" aria-hidden="true" /><h2 id="incidents-title">Recent incidents</h2></div>
          <LoadingStatus v-if="loadingIncidents && !incidents">Loading incidents…</LoadingStatus>
          <div v-if="incidentsError" class="console-notice danger" role="alert">
            {{ incidentsError.message }} <button type="button" class="text-action" @click="refreshIncidents">Retry incidents</button>
          </div>
          <PaginatedList v-if="incidents?.length" v-slot="{ items }" :items="incidents" :page-size="5" label="recent incidents">
            <ul class="incident-list">
              <li v-for="incident in items" :key="incident.id"><strong>{{ incident.cause }}</strong>
                <p>Started {{ formatDate(incident.startedAt) }}</p>
                <p v-if="incident.resolvedAt">Resolved {{ formatDate(incident.resolvedAt) }}</p><p v-else class="text-vigil-down">Ongoing</p>
              </li>
            </ul>
          </PaginatedList>
          <p v-else-if="incidents" class="history-empty">No incidents recorded.</p>
        </section>
      </div>
    </section>

    <dialog ref="editDialog" class="w-[min(38rem,calc(100%-2rem))] overflow-y-auto border border-vigil-rule bg-vigil-surface p-5 text-vigil-ink sm:p-6" aria-labelledby="edit-monitor-title">
      <div class="dialog-head"><h2 id="edit-monitor-title">Edit monitor</h2><button type="button" class="icon-control" aria-label="Close monitor editor" @click="editDialog?.close()"><X :size="18" /></button></div>
      <p v-if="editError" class="console-notice danger" role="alert">{{ editError }}</p>
      <MonitorForm v-if="monitor" :key="formVersion" :initial-value="monitor" :disabled="!!pending" :field-errors="fieldErrors" submit-label="Save changes" @submit="saveEdit" />
    </dialog>

    <ConfirmDialog
      :open="deleteOpen"
      title="Delete monitor?"
      :message="`Delete ${monitor?.name ?? 'this monitor'} and its check history? This cannot be undone.`"
      confirm-label="Delete monitor"
      :busy="!!pending"
      @cancel="deleteOpen = false"
      @confirm="confirmDelete"
    />
  </WorkspaceShell>
</template>
