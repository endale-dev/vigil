<script setup lang="ts">
import { computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ApiError } from '../api/client'
import { getPublicStatus } from '../api/status'
import type { PublicCheckDTO, UptimeSummaryDTO } from '../api/types'
import GuestShell from '../components/GuestShell.vue'
import StatusBadge from '../components/StatusBadge.vue'
import WorkspaceShell from '../components/WorkspaceShell.vue'
import { usePolling } from '../composables/usePolling'
import { useSession } from '../composables/useSession'

const route = useRoute()
const session = useSession()
const layoutComponent = computed(() => session.isReady.value && session.user.value ? WorkspaceShell : GuestShell)
const layoutProps = computed(() => layoutComponent.value === GuestShell
  ? { headerLinkLabel: 'Sign in', headerLinkTo: { name: 'login' } }
  : {})
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
  <component :is="layoutComponent" v-bind="layoutProps">
    <div v-if="!data && isLoading" class="mt-8 text-sm text-vigil-muted" role="status">Loading status page…</div>

    <section v-else-if="!data && loadError" class="mt-8 rounded-lg border border-vigil-rule bg-vigil-surface p-6">
      <p class="text-sm text-vigil-down" role="alert">{{ loadError }}</p>
      <button class="mt-4 min-h-10 rounded-md border border-vigil-rule px-3 py-2 text-sm hover:bg-vigil-paper focus-visible:outline-2 focus-visible:outline-offset-2" type="button" @click="refresh">Try again</button>
    </section>

    <div v-else-if="data" class="grid gap-6">
      <header class="border-b border-vigil-rule pb-6">
        <p class="text-sm font-medium text-vigil-brand">Service status</p>
        <h1 class="mt-2 text-3xl font-semibold tracking-tight text-vigil-ink">System status</h1>
        <p class="mt-2 text-sm text-vigil-muted">Updates every 30 seconds</p>
      </header>
      <section class="flex flex-wrap items-center justify-between gap-4 border-b border-vigil-rule py-4" aria-labelledby="overall-status-title">
        <h2 id="overall-status-title" class="text-sm font-medium text-vigil-muted">Overall status</h2>
        <StatusBadge :status="data.status" />
      </section>

      <section v-if="data.monitors.length" class="grid gap-4" aria-label="Public monitors">
        <article v-for="(monitor, monitorIndex) in data.monitors" :key="`${monitor.name}-${monitorIndex}`" class="border-b border-vigil-rule py-5">
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h3 class="text-lg font-semibold">{{ monitor.name }}</h3>
              <p class="mt-1 text-sm text-vigil-muted">{{ monitor.type === 'tls' ? 'TLS certificate' : monitor.type.toUpperCase() }}</p>
            </div>
            <StatusBadge :status="monitor.status" />
          </div>

          <dl class="mt-5 grid gap-4 sm:grid-cols-4">
            <div>
              <dt class="text-xs text-vigil-muted">Last check</dt>
              <dd class="mt-1 text-sm">{{ formatDate(monitor.lastCheckedAt) }}</dd>
            </div>
            <div>
              <dt class="text-xs text-vigil-muted">24-hour uptime</dt>
              <dd class="mt-1 text-sm font-medium">{{ uptimeLabel(monitor.uptime.last24Hours) }}</dd>
            </div>
            <div>
              <dt class="text-xs text-vigil-muted">7-day uptime</dt>
              <dd class="mt-1 text-sm font-medium">{{ uptimeLabel(monitor.uptime.last7Days) }}</dd>
            </div>
            <div>
              <dt class="text-xs text-vigil-muted">30-day uptime</dt>
              <dd class="mt-1 text-sm font-medium">{{ uptimeLabel(monitor.uptime.last30Days) }}</dd>
            </div>
          </dl>

          <section class="mt-6" aria-label="Recent check history">
            <h4 class="text-sm font-medium">Last 30 checks</h4>
            <ol v-if="monitor.checks.length" class="mt-3 flex flex-wrap gap-1" aria-label="Check history">
              <li v-for="(check, index) in monitor.checks" :key="index" :title="checkLabel(check)">
                <span
                  class="block h-6 w-2.5 rounded-sm"
                  :class="check.up ? 'bg-vigil-up' : 'bg-vigil-down'"
                  :aria-label="checkLabel(check)"
                  role="img"
                />
              </li>
            </ol>
            <p v-else class="mt-2 text-sm text-vigil-muted">No checks yet.</p>
          </section>
        </article>
      </section>

      <p v-else class="rounded-lg border border-vigil-rule bg-vigil-surface p-6 text-sm text-vigil-muted">No public monitors are available yet.</p>
      <p v-if="error" class="text-sm text-vigil-warn" role="status">Refresh failed. Showing the last loaded status.</p>
    </div>
  </component>
</template>
