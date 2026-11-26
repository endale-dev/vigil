<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { Check, ChevronDown, Filter, Pause, Play, Plus, RefreshCw, Search, Trash2, X } from '@lucide/vue'
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
import LoadingStatus from '../components/LoadingStatus.vue'
import MonitorForm from '../components/MonitorForm.vue'
import PaginatedList from '../components/PaginatedList.vue'
import StatusBadge from '../components/StatusBadge.vue'
import WorkspaceShell from '../components/WorkspaceShell.vue'
import { checkIntervalLabel } from '../format'
import { usePolling } from '../composables/usePolling'

const {
  data: monitorData,
  error: monitorError,
  isLoading,
  refresh,
} = usePolling(listMonitors, 15_000)

const monitors = computed(() => monitorData.value ?? [])
const isEmpty = computed(() => monitorData.value !== null && !monitorError.value && !monitors.value.length)
const search = ref('')
const router = useRouter()
function openMonitorRow(event: MouseEvent, id: string): void {
  if ((event.target as Element).closest('a,button,input') || window.getSelection()?.type === 'Range') return
  const route = { name: 'monitor-detail', params: { id } }
  if (event.ctrlKey || event.metaKey || event.button === 1) {
    window.open(router.resolve(route).href, '_blank', 'noopener')
  } else if (event.button === 0) {
    void router.push(route)
  }
}
type MonitorStatusFilter = 'all' | 'up' | 'down' | 'unknown' | 'paused'
const statusFilter = ref<MonitorStatusFilter>('all')
const pageFilters = computed(() => [search.value, statusFilter.value])
const statusFilterRoot = ref<HTMLElement | null>(null)
const statusFilterTrigger = ref<HTMLButtonElement | null>(null)
const statusFilterOpen = ref(false)
const statusFilterOptionButtons = ref<HTMLButtonElement[]>([])
const lastSynced = ref('')
watch(monitorData, () => {
  lastSynced.value = new Date().toLocaleTimeString(undefined, { hour: 'numeric', minute: '2-digit' })
})
const filteredMonitors = computed(() => {
  const term = search.value.trim().toLowerCase()
  return monitors.value.filter(monitor => {
    const state = monitor.enabled ? monitor.status : 'paused'
    return (statusFilter.value === 'all' || statusFilter.value === state)
      && [monitor.name, monitor.target, monitor.type].some(value => value.toLowerCase().includes(term))
  })
})
const statusFilterOptions = computed(() => {
  const counts: Record<MonitorStatusFilter, number> = { all: monitors.value.length, up: 0, down: 0, unknown: 0, paused: 0 }
  monitors.value.forEach(monitor => {
    counts[monitor.enabled ? monitor.status as Exclude<MonitorStatusFilter, 'all'> : 'paused']++
  })
  const options: { value: MonitorStatusFilter; label: string }[] = [
    { value: 'all', label: 'All statuses' },
    { value: 'up', label: 'Up' },
    { value: 'down', label: 'Down' },
    { value: 'unknown', label: 'Unknown' },
    { value: 'paused', label: 'Paused' },
  ]
  return options.map(option => ({ ...option, count: counts[option.value] }))
})
const selectedStatusFilterLabel = computed(() => statusFilterOptions.value.find(option => option.value === statusFilter.value)?.label ?? 'All statuses')
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
  if (activeMonitors.value.length === 0) {
    return { title: 'All monitors paused', tone: 'neutral' }
  }
  if (downCount.value > 0) {
    return { title: `${downCount.value} ${downCount.value === 1 ? 'monitor is' : 'monitors are'} down`, tone: 'danger' }
  }
  if (unknownCount.value > 0) {
    return { title: 'Waiting for initial checks', tone: 'neutral' }
  }
  return { title: 'All systems operational', tone: 'success' }
})
const createDialog = ref<HTMLDialogElement | null>(null)
const creating = ref(false)
const createError = ref('')
const formErrors = ref<Record<string, string | undefined>>({})
const actionError = ref('')
const pendingActionID = ref<string | null>(null)
const deleteTarget = ref<MonitorListDTO | null>(null)

function focusStatusFilterOption(index: number): void {
  statusFilterOptionButtons.value[index]?.focus()
}

function setStatusFilterOptionButton(element: unknown, index: number): void {
  if (element instanceof HTMLButtonElement) {
    statusFilterOptionButtons.value[index] = element
  }
}

async function openStatusFilter(focusIndex?: number): Promise<void> {
  statusFilterOpen.value = true
  await nextTick()
  const selectedIndex = statusFilterOptions.value.findIndex(option => option.value === statusFilter.value)
  focusStatusFilterOption(focusIndex ?? Math.max(selectedIndex, 0))
}

function closeStatusFilter(restoreFocus = false): void {
  statusFilterOpen.value = false
  if (restoreFocus) {
    nextTick(() => statusFilterTrigger.value?.focus())
  }
}

function selectStatusFilter(value: MonitorStatusFilter): void {
  statusFilter.value = value
  closeStatusFilter(true)
}

function handleStatusFilterKeydown(event: KeyboardEvent, index: number): void {
  let nextIndex: number | null = null
  if (event.key === 'ArrowDown') nextIndex = (index + 1) % statusFilterOptions.value.length
  if (event.key === 'ArrowUp') nextIndex = (index - 1 + statusFilterOptions.value.length) % statusFilterOptions.value.length
  if (event.key === 'Home') nextIndex = 0
  if (event.key === 'End') nextIndex = statusFilterOptions.value.length - 1
  if (event.key === 'Escape') {
    event.preventDefault()
    closeStatusFilter(true)
    return
  }
  if (nextIndex !== null) {
    event.preventDefault()
    focusStatusFilterOption(nextIndex)
  }
}

function handleStatusFilterPointer(event: PointerEvent): void {
  if (statusFilterOpen.value && !statusFilterRoot.value?.contains(event.target as Node)) {
    closeStatusFilter()
  }
}

onMounted(() => document.addEventListener('pointerdown', handleStatusFilterPointer))
onBeforeUnmount(() => document.removeEventListener('pointerdown', handleStatusFilterPointer))

function errorMessage(error: unknown, fallback: string): string {
  return error instanceof Error ? error.message : fallback
}

const formVersion = ref(0)
async function openCreateDialog(): Promise<void> {
  formVersion.value++
  createError.value = ''
  formErrors.value = {}
  await nextTick()
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
    <section aria-labelledby="overview-title">
      <header class="console-page-head">
        <div>
          <h1 id="overview-title" class="console-title">Overview</h1>
          <p v-if="monitors.length && !monitorError" class="system-health" :data-tone="overallStatus.tone" role="status">
            {{ overallStatus.title }}
          </p>
        </div>
        <button v-if="!isEmpty" type="button" class="console-button primary" @click="openCreateDialog">
          <Plus :size="16" aria-hidden="true" />
          Add monitor
        </button>
      </header>

      <dl v-if="monitors.length" class="metric-grid" aria-label="Monitor summary">
        <div class="metric">
          <dt>Active monitors</dt>
          <dd>{{ activeMonitors.length }} <small v-if="pausedCount">{{ pausedCount }} paused</small></dd>
        </div>
        <div class="metric"><dt>Up</dt><dd>{{ upCount }}</dd></div>
        <div class="metric"><dt>Down</dt><dd :class="{ 'text-vigil-down': downCount }">{{ downCount }}</dd></div>
        <div class="metric">
          <dt>24h uptime</dt>
          <dd v-if="uptime24Hours !== null">{{ uptimeLabel(uptime24Hours) }}</dd>
          <dd v-else><span aria-hidden="true">—</span><span class="sr-only">Awaiting checks</span></dd>
        </div>
      </dl>

      <section class="monitor-panel console-panel" aria-labelledby="monitor-list-title">
        <div class="monitor-heading">
          <h2 id="monitor-list-title" class="section-title">Monitors</h2>
          <div v-if="monitors.length" class="monitor-toolbar">
            <label class="search-control"><Search :size="15" aria-hidden="true" /><input v-model="search" aria-label="Search monitors" type="search" placeholder="Search monitors…" /></label>
            <div ref="statusFilterRoot" class="status-filter">
              <button ref="statusFilterTrigger" type="button" class="status-filter-trigger" aria-label="Filter monitors by status"
                aria-haspopup="menu" :aria-expanded="statusFilterOpen" aria-controls="monitor-status-filter-menu"
                @click="statusFilterOpen ? closeStatusFilter() : openStatusFilter()"
                @keydown.down.prevent="openStatusFilter()">
                <Filter :size="14" aria-hidden="true" />
                <span>{{ selectedStatusFilterLabel }}</span>
                <ChevronDown :size="14" class="status-filter-chevron" aria-hidden="true" />
              </button>
              <div v-if="statusFilterOpen" ref="statusFilterMenu" id="monitor-status-filter-menu" class="status-filter-menu" role="menu" aria-label="Filter monitors by status">
                <p class="status-filter-menu-heading">Monitor status</p>
                <button v-for="(option, index) in statusFilterOptions" :key="option.value" :ref="element => setStatusFilterOptionButton(element, index)"
                  type="button" role="menuitemradio" class="status-filter-option" :aria-checked="statusFilter === option.value" :tabindex="statusFilter === option.value ? 0 : -1"
                  @click="selectStatusFilter(option.value)" @keydown="handleStatusFilterKeydown($event, index)">
                  <span v-if="option.value !== 'all'" class="status-filter-dot" :data-status="option.value" aria-hidden="true"></span>
                  <span v-else class="status-filter-all" aria-hidden="true"><Filter :size="13" /></span>
                  <span class="status-filter-option-label">{{ option.label }}</span>
                  <span class="status-filter-count">{{ option.count }}</span>
                  <Check v-if="statusFilter === option.value" :size="15" class="status-filter-check" aria-hidden="true" />
                </button>
              </div>
            </div>
          </div>
        </div>
        <p v-if="actionError" class="console-notice danger" role="alert">{{ actionError }}</p>
        <div v-if="monitorError" class="console-notice danger" role="alert">
          {{ monitorData ? 'Showing the last available results.' : '' }} {{ loadError }}
          <button type="button" class="text-action" :disabled="isLoading" @click="refresh">{{ isLoading ? 'Retrying…' : 'Try again' }}</button>
        </div>
        <div v-if="!monitorData && !monitorError" class="monitor-empty"><LoadingStatus>Loading monitors…</LoadingStatus></div>
        <div v-if="isEmpty" class="monitor-empty first-monitor">
          <h3>No monitors yet</h3>
          <p>Add an HTTP, TCP, or TLS endpoint to start checking its availability.</p>
          <button type="button" class="console-button primary" @click="openCreateDialog">
            <Plus :size="16" aria-hidden="true" />
            Add monitor
          </button>
        </div>
        <PaginatedList v-if="monitors.length" :items="filteredMonitors" :page-size="10" label="monitors" :reset-on="pageFilters">
          <template #default="{ items }">
            <table v-if="filteredMonitors.length" class="monitor-table" aria-label="Your monitors">
              <thead><tr><th scope="col">Service</th><th scope="col">Status</th><th scope="col">24h uptime</th><th scope="col">Last check</th><th scope="col"><span class="sr-only">Actions</span></th></tr></thead>
              <tbody><tr v-for="monitor in items" :key="monitor.id" class="monitor-row" @click="openMonitorRow($event, monitor.id)" @auxclick="openMonitorRow($event, monitor.id)">
                <th scope="row"><div class="service-text">
                  <RouterLink :to="{ name: 'monitor-detail', params: { id: monitor.id } }">{{ monitor.name }}</RouterLink>
                  <small :title="monitor.target">{{ monitor.type.toUpperCase() }} · {{ monitor.target }}</small>
                </div></th>
                <td class="status-cell"><StatusBadge plain :status="monitor.enabled ? monitor.status : 'paused'" /></td>
                <td class="uptime-cell"><span class="cell-label">24h uptime</span><span class="number">{{ monitor.uptime24Hours.ratio === null ? '—' : uptimeLabel(monitor.uptime24Hours.ratio) }}</span>
                </td>
                <td class="last-cell"><span class="cell-label">Last check</span>{{ lastCheckedLabel(monitor.lastCheckedAt) }}<small>{{ checkIntervalLabel(monitor.intervalSeconds) }}</small></td>
                <td class="actions-cell"><div>
                  <button type="button" class="icon-control" :aria-label="(monitor.enabled ? 'Pause ' : 'Resume ') + monitor.name" :title="monitor.enabled ? 'Pause monitor' : 'Resume monitor'" :disabled="pendingActionID !== null" @click="toggleMonitor(monitor)"><component :is="monitor.enabled ? Pause : Play" :size="15" /></button>
                  <button type="button" class="icon-control destructive" :aria-label="'Delete ' + monitor.name" title="Delete monitor" :disabled="pendingActionID !== null" @click="deleteTarget = monitor"><Trash2 :size="15" /></button>
                </div></td>
              </tr></tbody>
            </table>
            <div v-else class="monitor-empty" role="status"><h3>No matching monitors</h3><button type="button" class="text-action" @click="search = ''; statusFilter = 'all'">Clear filters</button></div>
          </template>
          <template #footer>
            <div class="table-sync">
              <span>{{ monitorError ? 'Last successful update' : 'Updated' }} {{ lastSynced }}</span>
              <button type="button" class="icon-control" aria-label="Refresh monitors" title="Refresh monitors (auto-refreshes every 15 seconds)" :disabled="isLoading" @click="refresh">
                <RefreshCw :size="14" :class="{ refreshing: isLoading }" aria-hidden="true" />
              </button>
            </div>
          </template>
        </PaginatedList>
      </section>
    </section>


    <dialog ref="createDialog" class="max-h-[90vh] w-[min(42rem,calc(100%-2rem))] overflow-y-auto rounded-lg border border-vigil-rule bg-vigil-surface p-6 text-vigil-ink shadow-xl backdrop:bg-vigil-ink/35 sm:p-7" aria-labelledby="create-monitor-title">
      <div class="dialog-head">
        <h2 id="create-monitor-title">Add monitor</h2>
        <button class="icon-control" type="button" aria-label="Close monitor form" title="Close" @click="closeCreateDialog"><X :size="18" /></button>
      </div>
      <p v-if="createError" class="mb-5 border-l-2 border-vigil-down bg-vigil-down-soft px-4 py-3 text-sm text-vigil-down" role="alert">
        {{ createError }}
      </p>
      <MonitorForm :key="formVersion" :disabled="creating" :field-errors="formErrors" submit-label="Create monitor" @submit="submitCreate" />
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
