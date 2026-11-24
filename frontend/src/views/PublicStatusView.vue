<script setup lang="ts">
import { computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ApiError } from '../api/client'
import { getPublicStatus } from '../api/status'
import type { PublicCheckDTO, UptimeSummaryDTO } from '../api/types'
import AppShell from '../components/AppShell.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { usePolling } from '../composables/usePolling'

const route = useRoute()
const slug = computed(() => typeof route.params.slug === 'string' ? route.params.slug : '')
const { data, error, isLoading, refresh } = usePolling(() => getPublicStatus(slug.value), 30_000)
const loadError = computed(() => {
  if (!error.value) return ''
  if (error.value instanceof ApiError && error.value.status === 404) return 'This status page could not be found.'
  return error.value instanceof Error ? error.value.message : 'Could not load this status page.'
})

watch(slug, () => void refresh())

function uptimeLabel(summary: UptimeSummaryDTO): string {
  return summary.ratio === null ? 'No data' : `${(summary.ratio * 100).toFixed(2)}%`
}

function formatDate(value: string | null): string {
  if (!value) return 'No checks yet'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? 'Time unavailable' : date.toLocaleString()
}

function checkLabel(check: PublicCheckDTO): string {
  return `${check.up ? 'Up' : 'Down'}, ${check.latencyMs} ms`
}
</script>

<template>
  <AppShell>
    <div v-if="!data && isLoading" class="mt-8 text-sm text-slate-300" role="status">Loading status page…</div>

    <section v-else-if="!data && loadError" class="mt-8 rounded-lg border border-slate-800 bg-slate-900 p-6">
      <p class="text-sm text-red-200" role="alert">{{ loadError }}</p>
      <button class="mt-4 min-h-10 rounded-md border border-slate-600 px-3 py-2 text-sm hover:bg-slate-800 focus-visible:outline-2 focus-visible:outline-offset-2" type="button" @click="refresh">Try again</button>
    </section>

    <div v-else-if="data" class="mt-8 grid gap-6">
      <section class="flex flex-wrap items-center justify-between gap-4 rounded-lg border border-slate-800 bg-slate-900 p-5 sm:p-6" aria-labelledby="overall-status-title">
        <div>
          <h2 id="overall-status-title" class="text-lg font-semibold">Overall status</h2>
          <p class="mt-1 text-sm text-slate-300">This page refreshes every 30 seconds.</p>
        </div>
        <StatusBadge :status="data.status" />
      </section>

      <section v-if="data.monitors.length" class="grid gap-4" aria-label="Public monitors">
        <article v-for="(monitor, monitorIndex) in data.monitors" :key="`${monitor.name}-${monitorIndex}`" class="rounded-lg border border-slate-800 bg-slate-900 p-5 sm:p-6">
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h3 class="text-lg font-semibold">{{ monitor.name }}</h3>
              <p class="mt-1 text-sm text-slate-400">{{ monitor.type === 'tls' ? 'TLS certificate' : monitor.type.toUpperCase() }}</p>
            </div>
            <StatusBadge :status="monitor.status" />
          </div>

          <dl class="mt-5 grid gap-4 sm:grid-cols-4">
            <div>
              <dt class="text-xs text-slate-400">Last check</dt>
              <dd class="mt-1 text-sm">{{ formatDate(monitor.lastCheckedAt) }}</dd>
            </div>
            <div>
              <dt class="text-xs text-slate-400">24-hour uptime</dt>
              <dd class="mt-1 text-sm font-medium">{{ uptimeLabel(monitor.uptime.last24Hours) }}</dd>
            </div>
            <div>
              <dt class="text-xs text-slate-400">7-day uptime</dt>
              <dd class="mt-1 text-sm font-medium">{{ uptimeLabel(monitor.uptime.last7Days) }}</dd>
            </div>
            <div>
              <dt class="text-xs text-slate-400">30-day uptime</dt>
              <dd class="mt-1 text-sm font-medium">{{ uptimeLabel(monitor.uptime.last30Days) }}</dd>
            </div>
          </dl>

          <section class="mt-6" aria-label="Recent check history">
            <h4 class="text-sm font-medium">Last 30 checks</h4>
            <ol v-if="monitor.checks.length" class="mt-3 flex flex-wrap gap-1" aria-label="Check history">
              <li v-for="(check, index) in monitor.checks" :key="index" :title="checkLabel(check)">
                <span
                  class="flex size-7 items-center justify-center rounded-sm text-xs font-semibold"
                  :class="check.up ? 'bg-emerald-950 text-emerald-200' : 'bg-red-950 text-red-200'"
                  :aria-label="checkLabel(check)"
                  role="img"
                >
                  <span aria-hidden="true">{{ check.up ? '✓' : '×' }}</span>
                </span>
              </li>
            </ol>
            <p v-else class="mt-2 text-sm text-slate-400">No checks yet.</p>
          </section>
        </article>
      </section>

      <p v-else class="rounded-lg border border-slate-800 bg-slate-900 p-6 text-sm text-slate-300">No public monitors are available yet.</p>
      <p v-if="error" class="text-sm text-amber-200" role="status">Refresh failed. Showing the last loaded status.</p>
    </div>
  </AppShell>
</template>
