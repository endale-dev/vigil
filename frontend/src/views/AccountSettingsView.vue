<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { Pencil } from '@lucide/vue'
import FormField from '../components/FormField.vue'
import { ApiError } from '../api/client'
import { updateProfile } from '../api/account'
import { changePassword } from '../api/auth'
import WorkspaceShell from '../components/WorkspaceShell.vue'
import { useSession } from '../composables/useSession'

const session = useSession()
const editButton = ref<HTMLButtonElement | null>(null)
const user = session.user
const editing = ref(false)
const name = ref('')
const saving = ref<'profile' | 'password' | null>(null)
const errorMessage = ref('')
const fieldError = ref('')
const savedMessage = ref('')
const currentPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const currentPasswordError = ref('')
const newPasswordError = ref('')
const confirmPasswordError = ref('')
const passwordMessage = ref('')
const passwordError = ref('')
const initials = computed(() => user.value?.name
  .split(/\s+/)
  .filter(Boolean)
  .slice(0, 2)
  .map((part) => part[0]?.toUpperCase())
  .join('') || 'V')
const signInMethod = computed(() => user.value?.googleLinked
  ? user.value.hasPassword ? 'Google and password' : 'Google'
  : 'Email and password')

watch(user, (value) => {
  if (value) name.value = value.name
}, { immediate: true })

function startEditing(): void {
  if (!user.value) return
  name.value = user.value.name
  errorMessage.value = ''
  fieldError.value = ''
  savedMessage.value = ''
  editing.value = true
  void nextTick(() => document.getElementById('profile-name')?.focus())
}

function cancelEditing(): void {
  name.value = user.value?.name ?? ''
  editing.value = false
  errorMessage.value = ''
  fieldError.value = ''
  void nextTick(() => editButton.value?.focus())
}

async function saveProfile(): Promise<void> {
  if (saving.value) return
  const displayName = name.value.trim()
  fieldError.value = ''
  errorMessage.value = ''
  savedMessage.value = ''
  if (!displayName || [...displayName].length > 80) {
    fieldError.value = 'Use a name between 1 and 80 characters.'
    return
  }

  saving.value = 'profile'
  try {
    await updateProfile(displayName)
    await session.refresh()
    editing.value = false
    savedMessage.value = 'Your profile is up to date.'
    void nextTick(() => editButton.value?.focus())
  } catch (error) {
    if (error instanceof ApiError && error.field === 'name') {
      fieldError.value = error.message
    } else {
      errorMessage.value = error instanceof Error ? error.message : 'Could not save your profile.'
    }
  } finally {
    saving.value = null
  }
}

async function savePassword(): Promise<void> {
  if (saving.value) return
  currentPasswordError.value = ''
  newPasswordError.value = ''
  confirmPasswordError.value = ''
  passwordMessage.value = ''
  passwordError.value = ''
  if (newPassword.value !== confirmPassword.value) {
    confirmPasswordError.value = 'The new passwords do not match.'
    return
  }
  const passwordBytes = new TextEncoder().encode(newPassword.value).length
  if (passwordBytes < 8 || passwordBytes > 128) {
    newPasswordError.value = 'Password must be between 8 and 128 bytes.'
    return
  }

  saving.value = 'password'
  try {
    await changePassword(currentPassword.value, newPassword.value)
    currentPassword.value = ''
    newPassword.value = ''
    confirmPassword.value = ''
    passwordMessage.value = 'Password changed.'
  } catch (error) {
    if (error instanceof ApiError && error.field === 'currentPassword') {
      currentPasswordError.value = error.message
    } else if (error instanceof ApiError && error.field === 'password') {
      newPasswordError.value = error.message
    } else {
      passwordError.value = error instanceof Error ? error.message : 'Could not change the password.'
    }
  } finally {
    saving.value = null
  }
}
</script>

<template>
  <WorkspaceShell>
    <div v-if="!user" class="border-y border-vigil-rule py-8 text-sm text-vigil-muted" role="status">Loading account…</div>
    <div v-else class="account-workspace">
      <header class="console-page-head"><h1 class="console-title">Account settings</h1></header>
      <section class="console-panel" aria-labelledby="profile-title">
        <h2 id="profile-title" class="section-title px-5 pt-5 sm:px-6 sm:pt-6">Profile</h2>
        <div v-if="!editing" class="identity-row">
          <span class="console-avatar large" aria-hidden="true">{{ initials }}</span>
          <div class="identity-name">
            <p>Display name</p>
            <div class="flex items-center gap-1">
              <strong class="min-w-0">{{ user.name }}</strong>
              <button ref="editButton" type="button" class="icon-control" aria-label="Edit display name" title="Edit display name"
                :disabled="saving !== null" @click="startEditing">
                <Pencil :size="17" aria-hidden="true" />
              </button>
            </div>
          </div>
        </div>
        <form v-else class="profile-edit" @submit.prevent="saveProfile" @keydown.esc="!saving && cancelEditing()">
          <FormField id="profile-name" v-model="name" label="Display name" autocomplete="name" :disabled="saving !== null"
            :error="fieldError" required @update:model-value="fieldError = ''" />
          <p v-if="errorMessage" class="text-sm text-vigil-down" role="alert">{{ errorMessage }}</p>
          <div class="flex gap-2">
            <button class="console-button primary" type="submit" :disabled="saving !== null">{{ saving === 'profile' ? 'Saving…' : 'Save name' }}</button>
            <button type="button" class="console-button secondary" :disabled="saving !== null" @click="cancelEditing">Cancel</button>
          </div>
        </form>
        <p v-if="savedMessage" class="profile-feedback" role="status">{{ savedMessage }}</p>
        <dl class="profile-details">
          <div><dt>Email address</dt><dd>{{ user.email }}</dd></div>
          <div><dt>Sign-in method</dt><dd>{{ signInMethod }}</dd></div>
        </dl>
      </section>
      <section class="console-panel settings-panel mt-6" aria-labelledby="security-title">
        <h2 id="security-title" class="section-title">Security</h2>
        <p v-if="!user.hasPassword" class="mt-3 text-sm text-vigil-muted">You sign in with Google. Manage your password through your Google account.</p>
        <form v-else class="mt-4 grid gap-4" aria-label="Change password" @submit.prevent="savePassword">
          <FormField id="current-password" v-model="currentPassword" label="Current password" type="password" autocomplete="current-password"
            revealable required :disabled="saving !== null" :error="currentPasswordError" @update:model-value="currentPasswordError = ''" />
          <FormField id="new-password" v-model="newPassword" label="New password" type="password" autocomplete="new-password"
            hint="Use 8 to 128 bytes." revealable required :disabled="saving !== null" :error="newPasswordError" @update:model-value="newPasswordError = ''" />
          <FormField id="confirm-password" v-model="confirmPassword" label="Confirm new password" type="password" autocomplete="new-password"
            revealable required :disabled="saving !== null" :error="confirmPasswordError" @update:model-value="confirmPasswordError = ''" />
          <p v-if="passwordError" class="text-sm text-vigil-down" role="alert">{{ passwordError }}</p>
          <p v-if="passwordMessage" class="text-sm text-vigil-up" role="status">{{ passwordMessage }}</p>
          <button class="console-button primary w-fit" type="submit" :disabled="saving !== null">
            {{ saving === 'password' ? 'Saving…' : 'Change password' }}
          </button>
        </form>
      </section>
    </div>
  </WorkspaceShell>
</template>
