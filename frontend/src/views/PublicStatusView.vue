<script setup lang="ts">
import { computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ApiError } from '../api/client'
import { getPublicStatus } from '../api/status'
import type { PublicCheckDTO, UptimeSummaryDTO } from '../api/types'
import GuestShell from '../components/GuestShell.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { usePolling } from '../composables/usePolling'

const route = useRoute()
const slug = computed(() => typeof route.params.slug === 'string' ? route.params.slug : '')
const { data, error, isLoading, refresh } = usePolling(() => getPublicStatus(slug.value), 30_000)
const loadError = computed(() => {
  if (!error.value) return ''
  if (error.value instanceof ApiError && error.value.status === 404) return 'This public page could not be found.'
  return error.value instanceof Error ? error.value.message : 'Could not load this public page.'
})
const monitorCounts = computed(() => {
  const monitors = data.value?.monitors ?? []
  return {
    total: monitors.length,
    up: monitors.filter((monitor) => monitor.status === 'up').length,
    down: monitors.filter((monitor) => monitor.status === 'down').length,
  }
})
const uptime24Hours = computed(() => {
  const summaries = data.value?.monitors.map((monitor) => monitor.uptime.last24Hours) ?? []
  const total = summaries.reduce((sum, summary) => sum + summary.total, 0)
  return total > 0
    ? summaries.reduce((sum, summary) => sum + summary.up, 0) / total
    : null
})
const latestCheckAt = computed(() => {
  const timestamps = data.value?.monitors
    .map((monitor) => monitor.lastCheckedAt)
    .filter((value): value is string => value !== null)
    .map((value) => new Date(value))
    .filter((date) => !Number.isNaN(date.getTime())) ?? []
  return timestamps.sort((left, right) => right.getTime() - left.getTime())[0] ?? null
})
const latestCheckLabel = computed(() => latestCheckAt.value
  ? `Last check ${formatDate(latestCheckAt.value.toISOString())}`
  : 'Waiting for the first check')
const overallSummary = computed(() => {
  if (!data.value) return { title: '', detail: '' }
  if (data.value.status === 'operational') {
    return { title: 'All systems operational', detail: `${monitorCounts.value.up} of ${monitorCounts.value.total} services are responding normally.` }
  }
  if (data.value.status === 'degraded') {
    return { title: 'Some services need attention', detail: `${monitorCounts.value.down} of ${monitorCounts.value.total} services are currently down.` }
  }
  const waiting = data.value.monitors.filter((monitor) => monitor.status === 'unknown').length
  return { title: 'Waiting for checks', detail: `${waiting} ${waiting === 1 ? 'service has' : 'services have'} not returned a result yet.` }
})

watch(slug, () => void refresh())

function uptimeLabel(summary: UptimeSummaryDTO): string {
  return summary.ratio === null ? 'Awaiting checks' : `${(summary.ratio * 100).toFixed(2)}%`
}

function formatDate(value: string | null): string {
  if (!value) return 'First check pending'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? 'Time unavailable' : date.toLocaleString()
}

function checkLabel(check: PublicCheckDTO): string {
  return `${check.up ? 'Up' : 'Down'}, ${check.latencyMs} ms`
}
</script>

<template>
  <GuestShell header-link-label="Sign in" :header-link-to="{ name: 'login' }">
    <div v-if="!data && isLoading" class="border-y border-vigil-rule py-8 text-sm text-vigil-muted" role="status">Loading public page…</div>

    <section v-else-if="!data && loadError" class="border-y border-vigil-rule py-6">
      <p class="text-sm text-vigil-down" role="alert">{{ loadError }}</p>
      <button class="mt-4 min-h-10 rounded-md border border-vigil-rule px-3 py-2 text-sm hover:bg-vigil-paper focus-visible:outline-2 focus-visible:outline-offset-2" type="button" @click="refresh">Try again</button>
    </section>

    <div v-else-if="data" class="grid w-full gap-0">
      <header class="flex flex-wrap items-end justify-between gap-4 border-b border-vigil-rule pb-6">
        <div>
          <h1 class="text-3xl font-semibold tracking-tight text-vigil-ink">Service status</h1>
          <p class="mt-2 text-sm text-vigil-muted">Current availability and recent check results for shared services.</p>
        </div>
      </header>

      <section v-if="monitorCounts.total" class="flex flex-wrap items-center justify-between gap-4 border-b border-vigil-rule py-5" aria-labelledby="overall-status-title">
        <div class="flex min-w-0 items-start gap-3">
          <span
            class="mt-1.5 size-2.5 shrink-0 rounded-full"
            :class="data.status === 'operational' ? 'bg-vigil-up' : data.status === 'degraded' ? 'bg-vigil-down' : 'bg-vigil-unknown'"
            aria-hidden="true"
          />
          <div>
            <h2 id="overall-status-title" class="font-semibold text-vigil-ink">{{ overallSummary.title }}</h2>
            <p class="mt-1 text-sm text-vigil-muted">{{ overallSummary.detail }}</p>
          </div>
        </div>
        <p class="text-xs text-vigil-muted">{{ latestCheckLabel }}</p>
      </section>

      <section v-if="monitorCounts.total" class="grid grid-cols-2 border-b border-vigil-rule sm:grid-cols-4" aria-label="Status summary">
        <div class="border-b border-vigil-rule py-4 pr-4 sm:border-b-0 sm:pr-6">
          <p class="text-sm text-vigil-muted">Services</p>
          <p class="mt-1 font-vigil-mono text-2xl tabular-nums">{{ monitorCounts.total }}</p>
        </div>
        <div class="border-b border-l border-vigil-rule py-4 pl-4 sm:border-b-0 sm:px-6">
          <p class="text-sm text-vigil-muted">Up</p>
          <p class="mt-1 font-vigil-mono text-2xl tabular-nums text-vigil-up">{{ monitorCounts.up }}</p>
        </div>
        <div class="py-4 pr-4 sm:border-l sm:border-vigil-rule sm:px-6">
          <p class="text-sm text-vigil-muted">Down</p>
          <p class="mt-1 font-vigil-mono text-2xl tabular-nums" :class="monitorCounts.down ? 'text-vigil-down' : 'text-vigil-ink'">{{ monitorCounts.down }}</p>
        </div>
        <div class="border-l border-vigil-rule py-4 pl-4 sm:pl-6">
          <p class="text-sm text-vigil-muted">24-hour uptime</p>
          <p v-if="uptime24Hours !== null" class="mt-1 font-vigil-mono text-2xl tabular-nums">{{ `${(uptime24Hours * 100).toFixed(2)}%` }}</p>
          <p v-else class="mt-2 text-sm font-medium text-vigil-muted">Awaiting checks</p>
        </div>
      </section>

      <section v-if="monitorCounts.total" class="pt-8" aria-labelledby="public-monitors-title">
        <div class="flex items-baseline justify-between gap-4 border-b border-vigil-rule pb-3">
          <h2 id="public-monitors-title" class="text-lg font-semibold tracking-tight">Shared services</h2>
        </div>

        <ul aria-labelledby="public-monitors-title">
          <li v-for="(monitor, monitorIndex) in data.monitors" :key="`${monitor.name}-${monitorIndex}`" class="border-b border-vigil-rule py-5">
            <div class="flex flex-wrap items-start justify-between gap-3">
              <div>
                <h3 class="font-semibold text-vigil-ink">{{ monitor.name }}</h3>
                <p class="mt-1 text-sm text-vigil-muted">{{ monitor.type === 'tls' ? 'TLS certificate' : monitor.type.toUpperCase() }}</p>
              </div>
              <StatusBadge :status="monitor.status" />
            </div>

            <dl class="mt-4 grid grid-cols-2 gap-x-5 gap-y-3 sm:grid-cols-4">
              <div>
                <dt class="text-xs text-vigil-muted">Last check</dt>
                <dd class="mt-1 text-sm">{{ formatDate(monitor.lastCheckedAt) }}</dd>
              </div>
              <div>
                <dt class="text-xs text-vigil-muted">24-hour uptime</dt>
                <dd class="mt-1 font-vigil-mono text-sm tabular-nums">{{ uptimeLabel(monitor.uptime.last24Hours) }}</dd>
              </div>
              <div>
                <dt class="text-xs text-vigil-muted">7-day uptime</dt>
                <dd class="mt-1 font-vigil-mono text-sm tabular-nums">{{ uptimeLabel(monitor.uptime.last7Days) }}</dd>
              </div>
              <div>
                <dt class="text-xs text-vigil-muted">30-day uptime</dt>
                <dd class="mt-1 font-vigil-mono text-sm tabular-nums">{{ uptimeLabel(monitor.uptime.last30Days) }}</dd>
              </div>
            </dl>

            <div class="mt-4 flex flex-wrap items-center gap-x-4 gap-y-2" aria-label="Recent check history">
              <p class="text-xs text-vigil-muted">Recent checks</p>
              <ol v-if="monitor.checks.length" class="flex flex-wrap gap-1" aria-label="Check history">
                <li v-for="(check, index) in monitor.checks" :key="index" :title="checkLabel(check)">
                  <span
                    class="block h-5 w-2.5 rounded-sm"
                    :class="check.up ? 'bg-vigil-up' : 'bg-vigil-down'"
                    :aria-label="checkLabel(check)"
                    role="img"
                  />
                </li>
              </ol>
              <span v-else class="text-xs text-vigil-muted">First check pending</span>
            </div>
          </li>
        </ul>

      </section>
      <section v-else class="border-b border-vigil-rule py-10 text-center" aria-labelledby="no-shared-services-title">
        <h2 id="no-shared-services-title" class="font-semibold text-vigil-ink">No services shared yet</h2>
      </section>
      <p v-if="error" class="text-sm text-vigil-warn" role="status">Refresh failed. Showing the last loaded status.</p>
    </div>
  </GuestShell>
</template>
