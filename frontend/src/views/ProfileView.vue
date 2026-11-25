<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ApiError } from '../api/client'
import { updateProfile } from '../api/account'
import WorkspaceShell from '../components/WorkspaceShell.vue'
import { useSession } from '../composables/useSession'

const session = useSession()
const user = session.user
const editing = ref(false)
const name = ref('')
const pending = ref(false)
const errorMessage = ref('')
const fieldError = ref('')
const savedMessage = ref('')
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
}

function cancelEditing(): void {
  name.value = user.value?.name ?? ''
  editing.value = false
  errorMessage.value = ''
  fieldError.value = ''
}

async function saveProfile(): Promise<void> {
  const displayName = name.value.trim()
  fieldError.value = ''
  errorMessage.value = ''
  savedMessage.value = ''
  if (!displayName || [...displayName].length > 80) {
    fieldError.value = 'Use a name between 1 and 80 characters.'
    return
  }

  pending.value = true
  try {
    await updateProfile(displayName)
    await session.refresh()
    editing.value = false
    savedMessage.value = 'Your profile is up to date.'
  } catch (error) {
    if (error instanceof ApiError && error.field === 'name') {
      fieldError.value = error.message
    } else {
      errorMessage.value = error instanceof Error ? error.message : 'Could not save your profile.'
    }
  } finally {
    pending.value = false
  }
}
</script>

<template>
  <WorkspaceShell>
    <div v-if="!user" class="border-y border-vigil-rule py-8 text-sm text-vigil-muted" role="status">Loading profile…</div>
    <div v-else class="mx-auto max-w-[760px] pt-4">
      <header class="border-b border-vigil-rule pb-6">
        <p class="text-sm font-medium text-vigil-brand">Your account</p>
        <h1 class="mt-2 text-3xl font-semibold tracking-tight">Profile</h1>
        <p class="mt-2 text-sm text-vigil-muted">Your identity and sign-in details.</p>
      </header>

      <section class="flex flex-wrap items-center justify-between gap-5 border-b border-vigil-rule py-7" aria-labelledby="profile-identity-title">
        <div class="flex min-w-0 items-center gap-4">
          <span class="grid size-14 shrink-0 place-items-center rounded-full border border-vigil-brand/30 bg-vigil-brand-soft text-base font-semibold text-vigil-brand" aria-hidden="true">{{ initials }}</span>
          <div class="min-w-0">
            <h2 id="profile-identity-title" class="truncate text-lg font-semibold">{{ user.name }}</h2>
            <p class="mt-1 break-all text-sm text-vigil-muted">{{ user.email }}</p>
          </div>
        </div>
        <button class="inline-flex min-h-10 items-center gap-2 rounded-md border border-vigil-rule px-3 text-sm font-medium hover:bg-vigil-surface focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-vigil-brand" type="button" @click="startEditing">
          Edit profile
        </button>
      </section>

      <form v-if="editing" class="grid max-w-lg gap-4 border-b border-vigil-rule py-6" @submit.prevent="saveProfile">
        <label for="profile-name" class="text-sm font-medium">Display name</label>
        <input id="profile-name" v-model="name" class="min-h-10 w-full rounded-md border border-vigil-rule bg-vigil-surface px-3 py-2 text-sm text-vigil-ink outline-none transition focus:border-vigil-brand focus:ring-2 focus:ring-vigil-brand/20 disabled:opacity-60" type="text" name="name" autocomplete="name" maxlength="80" :disabled="pending" required>
        <p class="text-xs text-vigil-muted">This name appears in your workspace.</p>
        <p v-if="fieldError" class="text-sm text-vigil-down" role="alert">{{ fieldError }}</p>
        <p v-if="errorMessage" class="text-sm text-vigil-down" role="alert">{{ errorMessage }}</p>
        <div class="flex flex-wrap items-center gap-2">
          <button class="min-h-10 rounded-md bg-vigil-brand px-4 text-sm font-semibold text-white hover:bg-vigil-brand-hover focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-vigil-brand disabled:opacity-60" type="submit" :disabled="pending">{{ pending ? 'Saving…' : 'Save changes' }}</button>
          <button class="min-h-10 rounded-md px-3 text-sm text-vigil-muted hover:bg-vigil-surface hover:text-vigil-ink focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-vigil-brand disabled:opacity-60" type="button" :disabled="pending" @click="cancelEditing">Cancel</button>
        </div>
      </form>
      <p v-if="savedMessage" class="border-b border-vigil-rule py-4 text-sm text-vigil-up" role="status">{{ savedMessage }}</p>

      <dl class="divide-y divide-vigil-rule">
        <div class="grid gap-1 py-5 sm:grid-cols-[12rem_minmax(0,1fr)] sm:gap-6">
          <dt class="text-sm text-vigil-muted">Email address</dt>
          <dd class="break-all text-sm font-medium">{{ user.email }}</dd>
        </div>
        <div class="grid gap-1 py-5 sm:grid-cols-[12rem_minmax(0,1fr)] sm:gap-6">
          <dt class="text-sm text-vigil-muted">Sign-in method</dt>
          <dd class="text-sm font-medium">{{ signInMethod }}</dd>
        </div>
      </dl>
    </div>
  </WorkspaceShell>
</template>
