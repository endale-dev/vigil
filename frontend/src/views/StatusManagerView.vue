<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import { ArrowUpRight, Copy, Globe, Link2, Pencil, Plus, Search, X } from '@lucide/vue'
import { RouterLink, useRouter } from 'vue-router'
import { updateStatusSlug } from '../api/account'
import { ApiError } from '../api/client'
import { getMonitor, listMonitors, updateMonitor } from '../api/monitors'
import type { MonitorListDTO } from '../api/types'
import FormField from '../components/FormField.vue'
import StatusBadge from '../components/StatusBadge.vue'
import WorkspaceShell from '../components/WorkspaceShell.vue'
import { usePolling } from '../composables/usePolling'
import { useSession } from '../composables/useSession'

const router = useRouter()
const session = useSession()
const statusSlug = computed(() => session.user.value?.statusSlug ?? '')
const publicPath = computed(() => statusSlug.value
  ? router.resolve({ name: 'public-status', params: { slug: statusSlug.value } }).href
  : '')
const publicUrl = computed(() => publicPath.value && new URL(publicPath.value, window.location.origin).href)
const { data: monitors, error: monitorError, refresh } = usePolling(listMonitors, 15_000)
const sharedMonitors = computed(() => monitors.value?.filter(monitor => monitor.public) ?? [])
const editorOpen = ref(false)
const saving = ref(false)
const addressEnding = ref('')
const fieldError = ref('')
const saveError = ref('')
const feedback = ref('')
const urlField = ref<HTMLInputElement | null>(null)
const editButton = ref<HTMLButtonElement | null>(null)
const picker = ref<HTMLDialogElement | null>(null)
const choices = ref<MonitorListDTO[]>([])
const selected = ref<string[]>([])
const search = ref('')
const publishing = ref(false)
const publishError = ref('')
const publishFeedback = ref('')
const visibleChoices = computed(() => choices.value.filter(monitor =>
  `${monitor.name} ${monitor.target}`.toLowerCase().includes(search.value.trim().toLowerCase())))
const selectionChanged = computed(() => choices.value.some(monitor => monitor.public !== selected.value.includes(monitor.id)))
const wasPublished = computed(() => choices.value.some(monitor => monitor.public))

function openPicker(): void {
  choices.value = (monitors.value ?? []).map(monitor => ({ ...monitor }))
  selected.value = choices.value.filter(monitor => monitor.public).map(monitor => monitor.id)
  search.value = ''
  publishError.value = ''
  picker.value?.showModal()
}

function closePicker(): void {
  if (!publishing.value) picker.value?.close()
}

async function publishSelection(): Promise<void> {
  if (!selectionChanged.value || publishing.value) return
  publishing.value = true
  publishError.value = ''
  publishFeedback.value = ''
  const changed = choices.value.filter(monitor => monitor.public !== selected.value.includes(monitor.id))
  const results = await Promise.allSettled(changed.map(async monitor => {
    // Read the current settings so publishing does not restore stale configuration.
    const current = await getMonitor(monitor.id)
    const isPublic = selected.value.includes(monitor.id)
    await updateMonitor(monitor.id, { name: current.name, type: current.type, target: current.target,
      intervalSeconds: current.intervalSeconds, timeoutSeconds: current.timeoutSeconds,
      expectedStatus: current.expectedStatus, tlsWarnDays: current.tlsWarnDays, public: isPublic })
    monitor.public = isPublic
  }))
  await refresh()
  publishing.value = false
  const failed = results.filter(result => result.status === 'rejected').length
  if (failed) {
    publishError.value = `${failed} monitor${failed === 1 ? '' : 's'} could not be updated. Any successful changes are saved. Try again to finish publishing.`
    return
  }
  picker.value?.close()
  publishFeedback.value = selected.value.length ? 'Your status page has been updated.' : 'All monitors have been removed from your status page.'
}

async function editAddress(): Promise<void> {
  editorOpen.value = true
  addressEnding.value = statusSlug.value
  feedback.value = ''
  fieldError.value = ''
  saveError.value = ''
  await nextTick()
  document.getElementById('status-address-ending')?.focus()
}

async function closeEditor(): Promise<void> {
  editorOpen.value = false
  await nextTick()
  editButton.value?.focus()
}

async function copyUrl(): Promise<void> {
  try {
    await navigator.clipboard.writeText(publicUrl.value)
    feedback.value = 'Public URL copied.'
  } catch {
    urlField.value?.select()
    feedback.value = 'URL selected. Use your browser’s copy command.'
  }
}

async function saveAddress(): Promise<void> {
  const address = addressEnding.value.trim()
  fieldError.value = ''
  saveError.value = ''
  if (!/^[a-z0-9][a-z0-9-]{1,38}[a-z0-9]$/.test(address)) {
    fieldError.value = 'Enter 3 to 40 lowercase letters, numbers, or hyphens. Start and end with a letter or number.'
    return
  }

  saving.value = true
  try {
    await updateStatusSlug(address)
    await session.refresh()
    await closeEditor()
    feedback.value = 'Public address updated.'
  } catch (error) {
    if (error instanceof ApiError && error.status === 409) {
      fieldError.value = 'This address is already in use.'
    } else {
      saveError.value = error instanceof Error ? error.message : 'Could not update the public address.'
    }
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <WorkspaceShell>
    <div class="account-workspace">
      <header class="console-page-head">
        <div>
          <h1 class="console-title">Status page</h1>
          <p class="console-description">Share service availability on a public page. Choose which monitors visitors can see.</p>
        </div>
      </header>

      <p v-if="publishFeedback" class="status-page-feedback" role="status">{{ publishFeedback }}</p>
      <div v-if="monitorError" class="console-notice danger" role="alert">
        Could not refresh your monitors. <button class="text-action" type="button" @click="refresh">Try again</button>
      </div>
      <p v-if="!monitors && !monitorError" class="console-description" role="status">Loading your status page…</p>
      <section v-if="monitors && !sharedMonitors.length" class="status-page-empty console-panel" aria-labelledby="status-empty-title">
        <span class="status-page-symbol" aria-hidden="true"><Globe :size="28" :stroke-width="1.6" /></span>
        <h2 id="status-empty-title">Your status page has no monitors yet</h2>
        <p v-if="monitors.length">Choose the monitors you want to share, then publish them to your public status page.</p>
        <p v-else>Add your first monitor from Overview, then return here to choose what to share on your public status page.</p>
        <button v-if="monitors.length" type="button" class="console-button primary" @click="openPicker"><Plus :size="16" aria-hidden="true" /> Select monitors</button>
        <RouterLink v-else class="console-button primary" :to="{ name: 'dashboard' }"><Plus :size="16" aria-hidden="true" /> Go to monitors</RouterLink>
      </section>

      <div v-if="sharedMonitors.length" class="grid gap-6">
        <section class="console-panel p-4 sm:p-6" aria-labelledby="public-url-title">
          <div class="flex flex-wrap items-center justify-between gap-2">
            <h2 id="public-url-title" class="section-title flex min-h-10 items-center gap-2"><Link2 :size="17" class="text-vigil-brand" aria-hidden="true" /> Public URL</h2>
            <button v-if="!editorOpen" ref="editButton" type="button" class="console-button secondary" @click="editAddress">
              <Pencil :size="14" aria-hidden="true" /> {{ statusSlug ? 'Edit URL' : 'Set up URL' }}
            </button>
          </div>
          <p class="mt-1 text-sm text-vigil-muted">Anyone with this address can open the page without signing in.</p>

          <div v-if="publicUrl" class="mt-4 grid grid-cols-[auto_1fr] gap-2 sm:grid-cols-[minmax(0,1fr)_auto_auto]">
            <input ref="urlField" aria-label="Current public URL" :value="publicUrl" readonly
              class="col-span-2 min-h-11 min-w-0 rounded-md border border-vigil-rule bg-vigil-paper px-3 font-vigil-mono text-xs sm:col-span-1 sm:text-sm" />
            <button type="button" class="console-button secondary whitespace-nowrap px-3!" @click="copyUrl"><Copy :size="15" aria-hidden="true" /> Copy URL</button>
            <a class="console-button primary whitespace-nowrap px-3!" :href="publicPath" target="_blank" rel="noopener" aria-label="View page in a new tab">
              View page <ArrowUpRight :size="15" aria-hidden="true" />
            </a>
          </div>
          <p v-if="feedback" class="mt-3 text-sm text-vigil-muted" role="status">{{ feedback }}</p>

          <form v-if="editorOpen" class="mt-5 grid gap-3 border-t border-vigil-rule pt-5" @submit.prevent="saveAddress" @keydown.esc="!saving && closeEditor()">
            <p v-if="statusSlug" class="text-sm text-vigil-muted">Changing the address makes the old link stop working.</p>
            <div class="max-w-sm">
              <FormField id="status-address-ending" v-model="addressEnding" label="Address ending" placeholder="e.g. my-services"
                hint="The part after /status/. Use 3 to 40 lowercase letters, numbers, or hyphens."
                :max-length="40" :disabled="saving" :error="fieldError" required @update:model-value="fieldError = ''" />
            </div>
            <p v-if="saveError" class="text-sm text-vigil-down" role="alert">{{ saveError }}</p>
            <div class="flex flex-wrap gap-2">
              <button class="console-button primary" type="submit" :disabled="saving || addressEnding.trim() === statusSlug">
                {{ saving ? 'Saving…' : 'Save URL' }}
              </button>
              <button class="console-button secondary" type="button" :disabled="saving" @click="closeEditor">Cancel</button>
            </div>
          </form>
        </section>

        <section class="console-panel" aria-labelledby="shared-monitors-title">
          <div class="flex flex-wrap items-center justify-between gap-3 p-5 sm:p-6">
            <div>
              <h2 id="shared-monitors-title" class="section-title">Shared monitors</h2>
              <p class="mt-1 text-sm text-vigil-muted">Visitors see monitor names, status, uptime, and recent checks.</p>
            </div>
            <button type="button" class="console-button secondary" @click="openPicker"><Pencil :size="14" aria-hidden="true" /> Edit selection</button>
          </div>
          <div v-if="monitorError" class="border-t border-vigil-rule p-5 text-sm sm:px-6" role="status">
            {{ monitors ? 'Could not refresh the list.' : 'Could not load monitors.' }}
            <button class="text-action ml-1" type="button" @click="refresh">Try again</button>
          </div>
          <p v-if="!monitors && !monitorError" class="border-t border-vigil-rule p-5 text-sm text-vigil-muted sm:px-6" role="status">Loading monitors…</p>
          <ul v-else-if="sharedMonitors.length" class="divide-y divide-vigil-rule border-t border-vigil-rule">
            <li v-for="monitor in sharedMonitors" :key="monitor.id" class="status-shared-row">
              <div class="status-monitor-text"><RouterLink :to="{ name: 'monitor-detail', params: { id: monitor.id } }">{{ monitor.name }}</RouterLink><small>{{ monitor.target }}</small></div>
              <StatusBadge plain :status="monitor.enabled ? monitor.status : 'paused'" />
            </li>
          </ul>
          <div v-else-if="monitors" class="border-t border-vigil-rule bg-vigil-paper p-5 sm:px-6">
            <p class="text-sm font-semibold">No monitors shared yet</p>
            <p class="mt-1 text-sm text-vigil-muted">Create or edit a monitor and select “Share on my public page.”</p>
          </div>
        </section>
      </div>
    </div>
    <dialog ref="picker" class="status-monitor-dialog" aria-labelledby="select-monitors-title" aria-describedby="select-monitors-description" @cancel="publishing && $event.preventDefault()">
      <div class="status-picker-head">
        <div><h2 id="select-monitors-title">Select monitors</h2><p id="select-monitors-description">Choose which monitors appear on your public status page.</p></div>
        <button type="button" class="icon-control" aria-label="Close monitor selection" :disabled="publishing" @click="closePicker"><X :size="18" aria-hidden="true" /></button>
      </div>
      <form @submit.prevent="publishSelection">
        <div class="status-picker-search"><label class="search-control"><Search :size="15" aria-hidden="true" /><input v-model="search" type="search" aria-label="Search available monitors" placeholder="Search monitors…" :disabled="publishing" /></label></div>
        <div class="status-picker-list">
          <label v-for="monitor in visibleChoices" :key="monitor.id" class="status-picker-row" :class="{ 'is-selected': selected.includes(monitor.id) }">
            <input v-model="selected" type="checkbox" :value="monitor.id" :disabled="publishing" />
            <span class="status-monitor-text"><strong>{{ monitor.name }}</strong><small>{{ monitor.target }}</small></span>
            <StatusBadge plain :status="monitor.enabled ? monitor.status : 'paused'" />
          </label>
          <p v-if="!visibleChoices.length" class="status-picker-no-results">No monitors match your search.</p>
        </div>
        <p class="status-picker-note">Visitors can see names, availability, uptime, and recent checks. Alerts stay private.</p>
        <p v-if="publishError" class="console-notice danger" role="alert">{{ publishError }}</p>
        <footer class="status-picker-footer">
          <span role="status">{{ selected.length }} selected</span>
          <div><button type="button" class="console-button secondary" :disabled="publishing" @click="closePicker">Cancel</button>
          <button type="submit" class="console-button primary" :disabled="publishing || !selectionChanged">{{ publishing ? 'Saving…' : !selected.length && wasPublished ? 'Remove from page' : wasPublished ? 'Save changes' : 'Publish status page' }}</button></div>
        </footer>
      </form>
    </dialog>
  </WorkspaceShell>
</template>
