<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import { Copy, Mail, Pencil, Send } from '@lucide/vue'
import { sendTestAlert, updateNotifications } from '../api/account'
import { ApiError } from '../api/client'
import type { NotificationChannel, NotifyDTO } from '../api/types'
import FormField from '../components/FormField.vue'
import LoadingStatus from '../components/LoadingStatus.vue'
import WorkspaceShell from '../components/WorkspaceShell.vue'
import { useSession } from '../composables/useSession'

const session = useSession()
const user = session.user
const channels = [
  { id: 'email', name: 'Email', icon: Mail },
  { id: 'telegram', name: 'Telegram', icon: Send },
] as const
const editing = ref<NotificationChannel | null>(null)
const draft = ref('')
const fieldError = ref('')
const startNonce = ref('')
const startPayload = ref('')
const personalLink = computed(() => startPayload.value ? officialBot + '?start=' + startPayload.value : '')
const copyStatus = ref('')
const telegramPlaceholder = '@yourname or t.me/yourgroup'
const officialBot = 'https://t.me/VigilMonitorAlertBot'

function hex(bytes: Uint8Array) {
  return Array.from(bytes, byte => byte.toString(16).padStart(2, '0')).join('')
}

async function preparePersonalLink(owner: string) {
  startPayload.value = ''
  copyStatus.value = ''
  if (!window.crypto?.subtle) return
  const storageKey = 'vigil.telegram.link.' + owner
  let nonce = sessionStorage.getItem(storageKey)
  if (!nonce || !/^[0-9a-f]{32}$/.test(nonce)) {
    nonce = hex(crypto.getRandomValues(new Uint8Array(16)))
    sessionStorage.setItem(storageKey, nonce)
  }
  const digest = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(owner + ':' + nonce))
  startNonce.value = nonce
  startPayload.value = 'vigil_' + hex(new Uint8Array(digest, 0, 16))
}

async function copyCommand() {
  const input = document.getElementById('telegram-link-command') as HTMLInputElement | null
  if (!input) return
  try {
    await navigator.clipboard.writeText(input.value)
    copyStatus.value = 'Copied. Send the command in your bot chat.'
  } catch {
    input.select()
    copyStatus.value = 'Command selected. Copy it and send it in your bot chat.'
  }
}
const pending = ref<'save' | 'test' | null>(null)
const feedback = ref<{ channel: NotificationChannel; text: string; failed?: boolean } | null>(null)
const hasChat = computed(() => Boolean(user.value?.notify.telegramChatId))
const canEdit = (id: NotificationChannel) => id === 'email' || user.value?.capabilities.telegram
function enabled(id: NotificationChannel) {
  const account = user.value
  if (!account) return false
  return id === 'email' ? account.notify.emailAlerts
    : account.capabilities.telegram && hasChat.value && !account.notify.telegramDisabled
}

async function edit(id: NotificationChannel) {
  if (!user.value) return
  draft.value = id === 'email' ? user.value.notify.emailDestination : user.value.notify.telegramChatId
  fieldError.value = ''
  feedback.value = null
  editing.value = id
  if (id === 'telegram') await preparePersonalLink(user.value.id)
  await nextTick()
  document.getElementById(id + '-destination')?.focus()
}

async function closeEditor(id: NotificationChannel) {
  editing.value = null
  await nextTick()
  document.getElementById('edit-' + id)?.focus()
}

async function persist(id: NotificationChannel, patch: Partial<NotifyDTO>, message: string) {
  if (!user.value || pending.value) return
  pending.value = 'save'
  feedback.value = null
  fieldError.value = ''
  try {
    const account = await updateNotifications({
      ...user.value.notify, ...patch,
      telegramStartNonce: editing.value === 'telegram' && patch.telegramChatId?.startsWith('@') ? startNonce.value : undefined,
    })
    session.applyNotifications(account)
    if (editing.value === 'telegram') sessionStorage.removeItem('vigil.telegram.link.' + account.id)
    feedback.value = { channel: id, text: message }
    pending.value = null
    if (editing.value === id) await closeEditor(id)
  } catch (err) {
    if (editing.value === id && err instanceof ApiError && err.status === 422) {
      fieldError.value = id === 'email' ? 'Enter a valid email address.' : err.message
    } else {
      feedback.value = { channel: id, text: 'Could not save your alert settings. Please try again.', failed: true }
    }
  } finally {
    pending.value = null
  }
}

function saveDestination(id: NotificationChannel) {
  const value = draft.value.trim()
  if (id === 'email') return persist(id, { emailDestination: value }, 'Email destination saved.')
  return persist(id, {
    telegramChatId: value,
    telegramDisabled: !value || (hasChat.value && !enabled(id)),
  }, !value ? 'Chat removed. Telegram alerts are off.' : hasChat.value ? 'Telegram destination saved.' : 'Chat saved. Telegram alerts are on.')
}

function toggle(id: NotificationChannel) {
  const on = !enabled(id)
  return persist(id, id === 'email' ? { emailAlerts: on } : { telegramDisabled: !on },
    'Alerts turned ' + (on ? 'on.' : 'off.'))
}

async function test(id: NotificationChannel) {
  if (pending.value || !enabled(id)) return
  pending.value = 'test'
  feedback.value = { channel: id, text: 'Requesting a test alert…' }
  try {
    await sendTestAlert(id)
    feedback.value = { channel: id, text: 'Test requested. Check your ' + (id === 'email' ? 'inbox.' : 'Telegram chat.') }
  } catch {
    feedback.value = { channel: id, failed: true, text: id === 'email'
      ? 'Could not send the test email. Check the destination and try again.'
      : 'Could not send the test message. Check the destination and make sure the bot can post in that chat.' }
  } finally {
    pending.value = null
  }
}
</script>

<template>
  <WorkspaceShell>
    <div class="account-workspace">
      <header class="console-page-head">
        <div>
          <h1 class="console-title">Alerts</h1>
          <p class="console-description">Choose where Vigil sends downtime and recovery alerts.</p>
        </div>
      </header>
      <LoadingStatus v-if="!user">Loading alert settings…</LoadingStatus>
      <div v-else class="grid gap-4">
        <section v-for="{ id, name, icon } in channels" :key="id" class="console-panel p-5 sm:p-6" :aria-labelledby="id + '-title'">
          <header class="flex items-center gap-3">
            <span class="channel-icon"><component :is="icon" :size="19" aria-hidden="true" /></span>
            <h2 :id="id + '-title'" class="section-title grow">{{ name }}</h2>
            <button v-if="canEdit(id) && (id === 'email' || hasChat)" type="button" role="switch"
              class="channel-switch" :aria-label="name + ' alerts'" :aria-checked="enabled(id)"
              :disabled="!!pending || !!editing" @click="toggle(id)">
              {{ enabled(id) ? 'On' : 'Off' }} <span aria-hidden="true"></span>
            </button>
          </header>
          <div v-if="id === 'telegram' && canEdit(id)" class="mt-4 space-y-2 text-sm text-vigil-muted">
            <p>Vigil sends alerts through @VigilMonitorAlertBot. <button v-if="!hasChat && !editing" type="button" class="text-action" :disabled="!!pending" @click="edit('telegram')">Connect your chat</button></p>
            <details class="telegram-guide">
              <summary>Telegram connection guide</summary>
              <div class="grid gap-4 pt-3">
                <section>
                  <h3>Receive alerts in a personal chat</h3>
                  <ol>
                    <li>Choose <strong>Connect your chat</strong>, or use the pencil to change an existing destination.</li>
                    <li>Choose <strong>Connect with @VigilMonitorAlertBot</strong> and tap <strong>Start</strong> in Telegram. If the chat is already open, you can use <strong>Already started the bot?</strong> in Vigil and send the full copied command there instead.</li>
                    <li>Return to Vigil and save your own Telegram <code>@username</code> within 15 minutes of sending the command.</li>
                  </ol>
                </section>
                <section>
                  <h3>Receive alerts in a group</h3>
                  <ol>
                    <li>Open the destination editor and choose <strong>Add the bot to your group</strong>. Give it permission to post messages.</li>
                    <li>For a public group, paste its <code>t.me</code> link. For a private group, enter its numeric chat ID, including the leading minus sign. Save the destination.</li>
                  </ol>
                </section>
                <p>Once Telegram alerts are <strong>On</strong>, select <strong>Send test message</strong> and check the conversation you chose.</p>
              </div>
            </details>
          </div>
          <p v-if="!canEdit(id)" class="mt-4 text-sm text-vigil-muted">Telegram is unavailable. Ask the server administrator to configure the bot and restart Vigil.</p>
          <form v-else-if="editing === id" class="channel-editor" @submit.prevent="saveDestination(id)">
            <FormField :id="id + '-destination'" v-model="draft" :label="id === 'email' ? 'Destination email' : 'Telegram destination'"
              :type="id === 'email' ? 'email' : 'text'" :autocomplete="id === 'email' ? 'email' : 'off'"
              :placeholder="id === 'email' ? user.email : telegramPlaceholder"
              :hint="id === 'email' ? 'Leave blank to use ' + user.email + '.' : 'Use your personal @username, a public group link, or a numeric chat ID.'"
              :error="fieldError" :required="id === 'telegram' && !hasChat" :disabled="!!pending" @update:model-value="fieldError = ''" />
            <div v-if="id === 'telegram'" class="grid gap-2 text-sm text-vigil-muted">
              <template v-if="personalLink">
                <p><strong>Personal chat.</strong> <a class="text-action" :href="personalLink" target="_blank" rel="noopener noreferrer">Connect with @VigilMonitorAlertBot</a> and tap <strong>Start</strong>. Return here and save your @username within 15 minutes.</p>
                <details :open="!!fieldError && draft.trim().startsWith('@')">
                  <summary class="cursor-pointer font-medium text-vigil-brand">Already started the bot?</summary>
                  <p class="my-2">Send this full command in your existing private chat with @VigilMonitorAlertBot, then save your username again.</p>
                  <div class="flex items-center gap-2">
                    <input id="telegram-link-command" :value="'/start ' + startPayload" readonly aria-label="Telegram linking command" class="min-w-0 flex-1 rounded border border-vigil-rule p-2 font-mono text-xs" @focus="($event.target as HTMLInputElement).select()" />
                    <button type="button" class="console-button secondary" @click="copyCommand"><Copy :size="14" aria-hidden="true" />Copy</button>
                  </div>
                  <p v-if="copyStatus" role="status" class="mt-1">{{ copyStatus }}</p>
                </details>
              </template>
              <p v-else><strong>Personal chat.</strong> Use a numeric chat ID or open Vigil over HTTPS to link your username.</p>
              <p><strong>Group chat.</strong> <a class="text-action" :href="officialBot + '?startgroup=true'" target="_blank" rel="noopener noreferrer">Add the bot to your group</a>, then paste the group’s public t.me link. For a private group, use its numeric chat ID, including the minus sign.</p>
            </div>
            <p v-if="id === 'telegram' && hasChat" class="text-xs text-vigil-muted">Clear the field and save to disconnect this chat.</p>
            <div class="flex flex-wrap gap-2">
              <button class="console-button primary" type="submit" :disabled="!!pending">{{ pending === 'save' ? 'Saving…' : id === 'telegram' && !hasChat ? 'Save and enable alerts' : 'Save' }}</button>
              <button class="console-button secondary" type="button" :disabled="!!pending" @click="closeEditor(id)">Cancel</button>
            </div>
          </form>
          <div v-else class="channel-destination">
            <div class="min-w-0 grow">
              <p class="text-xs text-vigil-muted">{{ id === 'email' ? 'Destination email' : 'Telegram chat' }}</p>
              <p class="mt-1 break-all text-sm font-medium">{{ id === 'email' ? user.notify.emailDestination || user.email : user.notify.telegramChatId || 'Choose a chat for alerts' }}</p>
            </div>
            <button :id="'edit-' + id" type="button" class="icon-control" :aria-label="'Edit ' + name + ' destination'" :title="'Edit ' + name + ' destination'"
              :disabled="!!pending || !!editing" @click="edit(id)"><Pencil :size="16" aria-hidden="true" /></button>
          </div>
          <button v-if="enabled(id) && editing !== id" class="channel-test" type="button" :disabled="!!pending || !!editing" @click="test(id)">
            Send test {{ id === 'email' ? 'email' : 'message' }}
          </button>
          <p v-if="feedback?.channel === id" class="mt-3 text-sm" :class="feedback.failed ? 'text-vigil-down' : 'text-vigil-up'" :role="feedback.failed ? 'alert' : 'status'">{{ feedback.text }}</p>
        </section>
      </div>
    </div>
  </WorkspaceShell>
</template>

<style scoped>
.telegram-guide { border-left: 2px solid var(--color-vigil-rule); padding-left: 12px; }
.telegram-guide summary { width: fit-content; cursor: pointer; color: var(--color-vigil-brand); font-weight: 550; }
.telegram-guide h3 { margin-bottom: 4px; color: var(--color-vigil-ink); font-weight: 600; }
.telegram-guide ol { display: grid; gap: 4px; padding-left: 1.4em; list-style: decimal; }
.channel-icon { display: grid; place-items: center; width: 38px; height: 38px; border-radius: 6px; color: var(--color-vigil-brand); background: var(--color-vigil-brand-soft); }
.channel-destination,.channel-editor { margin-top: 16px; padding-top: 16px; border-top: 1px solid var(--color-vigil-rule); }
.channel-destination { display: flex; align-items: center; gap: 12px; }
.channel-editor { display: grid; gap: 14px; }
.channel-switch,.channel-test { display: inline-flex; align-items: center; min-height: 36px; font-size: 13px; font-weight: 550; }
.channel-test { margin-top: 8px; color: var(--color-vigil-brand); }
.channel-test:hover { text-decoration: underline; }
.channel-switch { gap: 8px; }
.channel-switch span { display: inline-flex; align-items: center; width: 34px; height: 20px; padding: 3px; border-radius: 20px; background: #69766e; }
.channel-switch span::after { content: ''; width: 14px; height: 14px; border-radius: 50%; background: white; }
.channel-switch[aria-checked=true] span { justify-content: flex-end; background: var(--color-vigil-brand); }
</style>
