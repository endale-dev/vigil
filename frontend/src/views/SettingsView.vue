<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ApiError } from '../api/client'
import { changePassword } from '../api/auth'
import { sendTestAlert, updateNotifications, updateStatusSlug } from '../api/account'
import type { NotificationChannel } from '../api/types'
import WorkspaceShell from '../components/WorkspaceShell.vue'
import FormField from '../components/FormField.vue'
import { useSession } from '../composables/useSession'

const session = useSession()
const user = session.user
const emailAlerts = ref(false)
const telegramChatId = ref('')
const statusSlug = ref('')
const currentPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const pending = ref(false)
const notificationsMessage = ref('')
const notificationsError = ref('')
const slugMessage = ref('')
const slugError = ref('')
const passwordMessage = ref('')
const passwordError = ref('')
const alertMessage = ref('')
const alertError = ref('')
const slugFieldError = ref('')
const profileMethod = computed(() => user.value?.googleLinked
  ? user.value.hasPassword ? 'Google and password' : 'Google'
  : 'Email and password')
const publicStatusURL = computed(() => user.value
  ? `${window.location.origin}/status/${user.value.statusSlug}`
  : '')

watch(user, (value) => {
  if (!value) return
  emailAlerts.value = value.notify.emailAlerts
  telegramChatId.value = value.notify.telegramChatId
  statusSlug.value = value.statusSlug
}, { immediate: true })

async function saveNotifications(): Promise<void> {
  pending.value = true
  notificationsMessage.value = ''
  notificationsError.value = ''
  try {
    await updateNotifications({ emailAlerts: emailAlerts.value, telegramChatId: telegramChatId.value.trim() })
    await session.refresh()
    notificationsMessage.value = 'Notification settings saved.'
  } catch (error) {
    notificationsError.value = error instanceof Error ? error.message : 'Could not save notification settings.'
  } finally {
    pending.value = false
  }
}

async function saveSlug(): Promise<void> {
  pending.value = true
  slugMessage.value = ''
  slugError.value = ''
  slugFieldError.value = ''
  if (!/^[a-z0-9](?:[a-z0-9-]{1,38}[a-z0-9])$/.test(statusSlug.value.trim())) {
    slugFieldError.value = 'Use 3 to 40 lowercase letters, numbers, or hyphens; do not start or end with a hyphen.'
    pending.value = false
    return
  }
  try {
    await updateStatusSlug(statusSlug.value.trim())
    await session.refresh()
    slugMessage.value = 'Public status address saved.'
  } catch (error) {
    if (error instanceof ApiError && (error.field === 'slug' || error.field === 'statusSlug')) {
      slugFieldError.value = error.message
    } else {
      slugError.value = error instanceof Error ? error.message : 'Could not save the public status address.'
    }
  } finally {
    pending.value = false
  }
}

async function savePassword(): Promise<void> {
  passwordMessage.value = ''
  passwordError.value = ''
  if (newPassword.value !== confirmPassword.value) {
    passwordError.value = 'The new passwords do not match.'
    return
  }
  if (new TextEncoder().encode(newPassword.value).length < 8 || new TextEncoder().encode(newPassword.value).length > 128) {
    passwordError.value = 'Password must be between 8 and 128 bytes.'
    return
  }

  pending.value = true
  try {
    await changePassword(currentPassword.value, newPassword.value)
    currentPassword.value = ''
    newPassword.value = ''
    confirmPassword.value = ''
    passwordMessage.value = 'Password changed.'
  } catch (error) {
    passwordError.value = error instanceof Error ? error.message : 'Could not change the password.'
  } finally {
    pending.value = false
  }
}

async function testAlert(channel: NotificationChannel): Promise<void> {
  pending.value = true
  alertMessage.value = ''
  alertError.value = ''
  try {
    await sendTestAlert(channel)
    alertMessage.value = `Test ${channel} alert sent.`
  } catch (error) {
    alertError.value = error instanceof Error ? error.message : 'Could not send the test alert.'
  } finally {
    pending.value = false
  }
}
</script>

<template>
  <WorkspaceShell>
    <div v-if="!user" class="mt-8 text-sm text-vigil-muted" role="status">Loading account…</div>
    <div v-else class="grid gap-8 pt-4">
      <header class="border-b border-vigil-rule pb-6">
        <p class="text-sm font-medium text-vigil-brand">Workspace</p>
        <h1 class="mt-2 text-3xl font-semibold tracking-tight">Settings</h1>
      </header>
      <section class="border-t border-vigil-rule pt-5" aria-labelledby="profile-title">
        <h2 id="profile-title" class="text-lg font-semibold">Profile</h2>
        <dl class="mt-4 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <div><dt class="text-sm text-vigil-muted">Name</dt><dd class="mt-1">{{ user.name }}</dd></div>
          <div><dt class="text-sm text-vigil-muted">Email</dt><dd class="mt-1 break-all">{{ user.email }}</dd></div>
          <div><dt class="text-sm text-vigil-muted">Sign-in method</dt><dd class="mt-1">{{ profileMethod }}</dd></div>
        </dl>
      </section>

      <section class="border-t border-vigil-rule pt-5" aria-labelledby="password-title">
        <h2 id="password-title" class="text-lg font-semibold">Password</h2>
        <p v-if="!user.hasPassword" class="mt-2 text-sm text-vigil-muted">This account signs in with Google and does not have a password.</p>
        <form v-else class="mt-4 grid max-w-xl gap-4" @submit.prevent="savePassword">
          <FormField id="current-password" v-model="currentPassword" label="Current password" type="password" autocomplete="current-password" :disabled="pending" required />
          <FormField id="new-password" v-model="newPassword" label="New password" type="password" autocomplete="new-password" :disabled="pending" required />
          <FormField id="confirm-password" v-model="confirmPassword" label="Confirm new password" type="password" autocomplete="new-password" :disabled="pending" required />
          <p class="text-xs text-vigil-muted">Use 8 to 128 bytes.</p>
          <p v-if="passwordError" class="text-sm text-vigil-down" role="alert">{{ passwordError }}</p>
          <p v-if="passwordMessage" class="text-sm text-vigil-up" role="status">{{ passwordMessage }}</p>
          <button class="min-h-10 w-fit rounded-md bg-vigil-brand px-4 py-2 text-sm font-medium text-white hover:bg-vigil-brand-hover focus-visible:outline-2 focus-visible:outline-offset-2 disabled:opacity-60" type="submit" :disabled="pending">{{ pending ? 'Saving…' : 'Change password' }}</button>
        </form>
      </section>

      <section class="border-t border-vigil-rule pt-5" aria-labelledby="notifications-title">
        <h2 id="notifications-title" class="text-lg font-semibold">Notifications</h2>
        <form class="mt-4 grid max-w-xl gap-4" @submit.prevent="saveNotifications">
          <label class="flex min-h-10 items-center gap-3 text-sm">
            <input v-model="emailAlerts" type="checkbox" class="size-4 accent-vigil-brand focus-visible:outline-2 focus-visible:outline-offset-2" :disabled="pending">
            Send alerts to {{ user.email }}
          </label>
          <FormField id="telegram-chat-id" v-model="telegramChatId" label="Telegram chat ID" :disabled="pending || !user.capabilities.telegram" placeholder="Your Telegram chat ID" :error="!user.capabilities.telegram ? 'Telegram notifications are not configured on this server.' : undefined" />
          <p v-if="notificationsError" class="text-sm text-vigil-down" role="alert">{{ notificationsError }}</p>
          <p v-if="notificationsMessage" class="text-sm text-vigil-up" role="status">{{ notificationsMessage }}</p>
          <button class="min-h-10 w-fit rounded-md bg-vigil-brand px-4 py-2 text-sm font-medium text-white hover:bg-vigil-brand-hover focus-visible:outline-2 focus-visible:outline-offset-2 disabled:opacity-60" type="submit" :disabled="pending">{{ pending ? 'Saving…' : 'Save notifications' }}</button>
        </form>
        <div class="mt-5 flex flex-wrap gap-3">
          <button class="min-h-10 rounded-md border border-vigil-rule px-3 py-2 text-sm hover:bg-vigil-paper focus-visible:outline-2 focus-visible:outline-offset-2 disabled:opacity-60" type="button" :disabled="pending || !user.notify.emailAlerts" @click="testAlert('email')">Send email test</button>
          <button class="min-h-10 rounded-md border border-vigil-rule px-3 py-2 text-sm hover:bg-vigil-paper focus-visible:outline-2 focus-visible:outline-offset-2 disabled:opacity-60" type="button" :disabled="pending || !user.capabilities.telegram || !user.notify.telegramChatId" @click="testAlert('telegram')">Send Telegram test</button>
        </div>
        <p v-if="alertError" class="mt-3 text-sm text-vigil-down" role="alert">{{ alertError }}</p>
        <p v-if="alertMessage" class="mt-3 text-sm text-vigil-up" role="status">{{ alertMessage }}</p>
      </section>

      <section class="border-t border-vigil-rule pt-5" aria-labelledby="public-status-title">
        <h2 id="public-status-title" class="text-lg font-semibold">Public status page</h2>
        <p class="mt-2 text-sm text-vigil-muted">Choose the address people can use to view your public monitors.</p>
        <form class="mt-4 grid max-w-xl gap-4" @submit.prevent="saveSlug">
          <FormField id="status-slug" v-model="statusSlug" label="Status page slug" :disabled="pending" :max-length="40" :error="slugFieldError" required />
          <p v-if="user.statusSlug" class="break-all text-sm text-vigil-muted">{{ publicStatusURL }}</p>
          <p v-if="slugError" class="text-sm text-vigil-down" role="alert">{{ slugError }}</p>
          <p v-if="slugMessage" class="text-sm text-vigil-up" role="status">{{ slugMessage }}</p>
          <button class="min-h-10 w-fit rounded-md bg-vigil-brand px-4 py-2 text-sm font-medium text-white hover:bg-vigil-brand-hover focus-visible:outline-2 focus-visible:outline-offset-2 disabled:opacity-60" type="submit" :disabled="pending">{{ pending ? 'Saving…' : 'Save status page' }}</button>
        </form>
      </section>
    </div>
  </WorkspaceShell>
</template>
