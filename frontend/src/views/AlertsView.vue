<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ApiError } from '../api/client'
import { sendTestAlert, updateNotifications } from '../api/account'
import type { NotificationChannel } from '../api/types'
import FormField from '../components/FormField.vue'
import WorkspaceShell from '../components/WorkspaceShell.vue'
import { useSession } from '../composables/useSession'

const session = useSession()
const user = session.user
const emailAlerts = ref(false)
const emailDestination = ref('')
const telegramAlerts = ref(false)
const telegramChatId = ref('')
const saving = ref(false)
const testing = ref<NotificationChannel | null>(null)
const saveError = ref('')
const emailError = ref('')
const telegramError = ref('')
const savedMessage = ref('')
const testResult = ref<{ channel: NotificationChannel; message: string; failed: boolean } | null>(null)

const savedTelegramEnabled = computed(() => Boolean(user.value?.notify.telegramChatId && !user.value.notify.telegramDisabled))

watch(user, (account) => {
  emailAlerts.value = account?.notify.emailAlerts ?? false
  emailDestination.value = account?.notify.emailDestination ?? ''
  telegramAlerts.value = savedTelegramEnabled.value
  telegramChatId.value = account?.notify.telegramChatId ?? ''
}, { immediate: true })

const emailChanged = computed(() => emailAlerts.value !== user.value?.notify.emailAlerts
  || emailDestination.value.trim() !== (user.value?.notify.emailDestination ?? ''))
const telegramChanged = computed(() => telegramAlerts.value !== savedTelegramEnabled.value
  || telegramChatId.value.trim() !== user.value?.notify.telegramChatId)
const hasChanges = computed(() => emailChanged.value || telegramChanged.value)
const busy = computed(() => saving.value || testing.value !== null)

async function save(): Promise<void> {
  if (!hasChanges.value || busy.value) return
  saving.value = true
  saveError.value = ''
  emailError.value = ''
  telegramError.value = ''
  savedMessage.value = ''
  testResult.value = null
  try {
    const chatId = telegramChatId.value.trim()
    await updateNotifications({
      emailAlerts: emailAlerts.value,
      emailDestination: emailDestination.value.trim(),
      telegramChatId: chatId,
      telegramDisabled: Boolean(chatId) && !telegramAlerts.value,
    })
    await session.refresh()
    savedMessage.value = 'Alert settings saved.'
  } catch (error) {
    if (error instanceof ApiError && error.field === 'emailDestination') {
      emailError.value = error.message
    } else if (error instanceof ApiError && error.field === 'telegramChatId') {
      telegramError.value = error.message
    } else {
      saveError.value = error instanceof Error ? error.message : 'Could not save alert settings.'
    }
  } finally {
    saving.value = false
  }
}

async function test(channel: NotificationChannel): Promise<void> {
  testing.value = channel
  testResult.value = null
  try {
    await sendTestAlert(channel)
    testResult.value = { channel, message: 'Test alert requested. Check your destination.', failed: false }
  } catch (error) {
    testResult.value = {
      channel,
      message: error instanceof Error ? error.message : 'Could not send the test alert.',
      failed: true,
    }
  } finally {
    testing.value = null
  }
}
</script>

<template>
  <WorkspaceShell>
    <div class="account-workspace">
      <header class="console-page-head">
        <div>
          <h1 class="console-title">Alerts</h1>
          <p class="console-description">Choose where Vigil sends downtime and recovery alerts for your monitors.</p>
        </div>
      </header>
      <p v-if="!user" role="status" class="text-sm text-vigil-muted">Loading alert settings…</p>
      <form v-else class="console-panel divide-y divide-vigil-rule" @submit.prevent="save">
        <section class="p-5 sm:p-6" aria-labelledby="email-alerts-title">
          <div class="flex flex-wrap items-center justify-between gap-2">
            <h2 id="email-alerts-title" class="section-title">Email</h2>
            <span class="text-sm" :class="user.notify.emailAlerts ? 'text-vigil-up' : 'text-vigil-muted'">
              {{ user.notify.emailAlerts ? 'Enabled' : 'Off' }}
            </span>
          </div>
          <label class="mt-2 flex min-h-10 items-center gap-2.5 text-sm">
            <input v-model="emailAlerts" type="checkbox" class="size-4 accent-vigil-brand" :disabled="busy" />
            Send email alerts
          </label>
          <div class="mt-2 max-w-md">
            <FormField id="alert-email" v-model="emailDestination" type="email" label="Destination email" autocomplete="email"
              :placeholder="user.email" :hint="`Leave blank to send alerts to your sign-in email (${user.email}).`"
              :disabled="busy" :error="emailError" @update:model-value="emailError = ''" />
          </div>
          <button v-if="user.notify.emailAlerts && !emailChanged" type="button" class="console-button secondary mt-2" :disabled="busy" @click="test('email')">
            {{ testing === 'email' ? 'Sending…' : 'Send test email' }}
          </button>
          <p v-if="testResult?.channel === 'email'" class="mt-2 text-sm" :class="testResult.failed ? 'text-vigil-down' : 'text-vigil-up'" :role="testResult.failed ? 'alert' : 'status'">
            {{ testResult.message }}
          </p>
        </section>

        <section class="p-5 sm:p-6" aria-labelledby="telegram-alerts-title">
          <div class="flex flex-wrap items-center justify-between gap-2">
            <h2 id="telegram-alerts-title" class="section-title">Telegram</h2>
            <span class="text-sm" :class="user.capabilities.telegram && savedTelegramEnabled ? 'text-vigil-up' : 'text-vigil-muted'">
              {{ !user.capabilities.telegram ? 'Unavailable' : savedTelegramEnabled ? 'Enabled' : 'Off' }}
            </span>
          </div>
          <template v-if="user.capabilities.telegram">
            <p class="mt-2 text-sm text-vigil-muted">
              Start a chat with your Vigil bot, or add it to the group that should get alerts.
            </p>
            <label class="mt-2 flex min-h-10 items-center gap-2.5 text-sm">
              <input v-model="telegramAlerts" type="checkbox" class="size-4 accent-vigil-brand" :disabled="busy" />
              Send Telegram alerts
            </label>
            <div class="mt-2 max-w-md">
              <FormField id="telegram-chat-id" v-model="telegramChatId" label="Telegram chat ID" autocomplete="off"
                hint="Use the numeric chat ID, including the minus sign if present."
                :required="telegramAlerts" :disabled="busy" :error="telegramError" @update:model-value="telegramError = ''" />
            </div>
            <details class="mt-3 text-sm text-vigil-muted">
              <summary class="w-fit cursor-pointer font-medium text-vigil-brand">Where do I find the chat ID?</summary>
              <p class="mt-2">Ask the person who configured the bot for the chat ID. If you manage the bot, find <code>message.chat.id</code> in the update for your message.</p>
              <a class="text-action mt-1 inline-block" href="https://core.telegram.org/bots/api#getting-updates" target="_blank" rel="noopener noreferrer">Telegram’s Bot API guide</a>
            </details>
            <button v-if="savedTelegramEnabled && !telegramChanged" type="button" class="console-button secondary mt-3" :disabled="busy" @click="test('telegram')">
              {{ testing === 'telegram' ? 'Sending…' : 'Send Telegram test' }}
            </button>
            <p v-if="testResult?.channel === 'telegram'" class="mt-2 text-sm" :class="testResult.failed ? 'text-vigil-down' : 'text-vigil-up'" :role="testResult.failed ? 'alert' : 'status'">
              {{ testResult.message }}
            </p>
          </template>
          <p v-else class="mt-2 text-sm text-vigil-muted">A server administrator needs to set <code>TELEGRAM_BOT_TOKEN</code> before you can connect a Telegram chat.</p>
        </section>

        <div class="flex flex-wrap items-center gap-x-4 gap-y-3 p-5 sm:px-6 sm:py-4">
          <button type="submit" class="console-button primary" :disabled="busy || !hasChanges">{{ saving ? 'Saving…' : 'Save alert settings' }}</button>
          <p v-if="saveError" role="alert" class="text-sm text-vigil-down">{{ saveError }}</p>
          <p v-else-if="savedMessage && !hasChanges" role="status" class="text-sm text-vigil-up">{{ savedMessage }}</p>
        </div>
      </form>
    </div>
  </WorkspaceShell>
</template>
