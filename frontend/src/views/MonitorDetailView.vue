<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { ArrowLeft, Trash2, X } from '@lucide/vue'
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
import type { CheckDTO, IncidentDTO, MonitorDetailDTO, MonitorRequest, UptimeSummaryDTO } from '../api/types'
import WorkspaceShell from '../components/WorkspaceShell.vue'
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
const formVersion = ref(0)
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
  <WorkspaceShell>
    <div v-if="loading" class="mt-8 text-sm text-vigil-muted" role="status">Loading monitor…</div>
    <section v-else-if="pageError" class="mt-8 rounded-lg border border-vigil-rule bg-vigil-surface p-6">
      <p class="text-sm text-vigil-down" role="alert">{{ pageError }}</p>
      <RouterLink class="mt-4 inline-block text-sm text-vigil-brand underline-offset-4 hover:underline focus-visible:outline-2" :to="{ name: 'dashboard' }">
        Back to dashboard
      </RouterLink>
    </section>

    <section v-else-if="monitor">
      <RouterLink class="detail-back" :to="{ name: 'dashboard' }"><ArrowLeft :size="15" /> All monitors</RouterLink>
      <p v-if="actionError" class="console-notice danger" role="alert">{{ actionError }}</p>
      <header class="console-page-head detail-head">
        <div class="min-w-0">
          <div class="flex flex-wrap items-center gap-3"><h1 class="console-title">{{ monitor.name }}</h1>
            <StatusBadge plain :status="monitor.enabled ? monitor.status : 'paused'" />
          </div>
          <p class="console-description break-all">{{ monitor.target }}</p>
        </div>
        <div class="detail-actions">
          <button class="console-button primary" type="button" :disabled="pending" @click="openEdit">Edit monitor</button>
          <button class="console-button secondary" type="button" :disabled="pending" @click="toggleMonitor">{{ pending ? 'Saving…' : monitor.enabled ? 'Pause' : 'Resume' }}</button>
          <button class="icon-control destructive" type="button" aria-label="Delete monitor" title="Delete monitor" :disabled="pending" @click="deleteOpen = true"><Trash2 :size="16" /></button>
        </div>
      </header>
      <dl class="detail-metadata console-panel">
        <div><dt>Monitor type</dt><dd>{{ typeLabel(monitor.type) }}</dd></div>
        <div><dt>Check interval</dt><dd>{{ checkIntervalLabel(monitor.intervalSeconds) }}</dd></div>
        <div><dt>Last check</dt><dd>{{ formatDate(monitor.lastCheckedAt) }}</dd></div>
      </dl>
      <section class="console-panel detail-uptime" aria-labelledby="uptime-title">
        <div class="panel-title"><h2 id="uptime-title">Uptime</h2></div>
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
          <div class="panel-title"><h2 id="checks-title">Recent checks</h2></div>
          <UptimeStrip v-if="checks.length || monitor.enabled" :checks="checks" />
          <p v-else class="history-empty">Resume this monitor to collect checks.</p>
          <ul v-if="checks.length" class="check-list">
            <li v-for="check in checks" :key="check.id">
              <div><span class="check-state" :class="{ failed: !check.up }">{{ check.up ? 'Up' : 'Down' }}</span> <span class="number">{{ check.latencyMs }} ms</span>
                <p v-if="check.detail">{{ check.detail }}</p>
              </div><time :datetime="check.checkedAt">{{ formatDate(check.checkedAt) }}</time>
            </li>
          </ul>
        </section>
        <section class="console-panel history-card" aria-labelledby="incidents-title">
          <div class="panel-title"><h2 id="incidents-title">Incidents</h2></div>
          <ul v-if="incidents.length" class="incident-list">
            <li v-for="incident in incidents" :key="incident.id"><strong>{{ incident.cause }}</strong>
              <p>Started {{ formatDate(incident.startedAt) }}</p>
              <p v-if="incident.resolvedAt">Resolved {{ formatDate(incident.resolvedAt) }}</p><p v-else class="text-vigil-down">Ongoing</p>
            </li>
          </ul><p v-else class="history-empty">No incidents recorded.</p>
        </section>
      </div>
    </section>

    <dialog ref="editDialog" class="w-[min(38rem,calc(100%-2rem))] overflow-y-auto border border-vigil-rule bg-vigil-surface p-5 text-vigil-ink sm:p-6" aria-labelledby="edit-monitor-title">
      <div class="dialog-head"><h2 id="edit-monitor-title">Edit monitor</h2><button type="button" class="icon-control" aria-label="Close monitor editor" @click="editDialog?.close()"><X :size="18" /></button></div>
      <p v-if="editError" class="console-notice danger" role="alert">{{ editError }}</p>
      <MonitorForm v-if="monitor" :key="formVersion" :initial-value="monitor" :disabled="pending" :field-errors="fieldErrors" submit-label="Save changes" @submit="saveEdit" />
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
  </WorkspaceShell>
</template>
