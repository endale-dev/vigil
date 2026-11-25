<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ApiError } from '../api/client'
import FormField from '../components/FormField.vue'
import GuestShell from '../components/GuestShell.vue'
import { useSession } from '../composables/useSession'

const router = useRouter()
const { register } = useSession()
const name = ref('')
const email = ref('')
const password = ref('')
const nameError = ref('')
const emailError = ref('')
const passwordError = ref('')
const errorMessage = ref('')
const submitting = ref(false)

async function submit(): Promise<void> {
  nameError.value = ''
  emailError.value = ''
  passwordError.value = ''
  errorMessage.value = ''

  if (!name.value.trim()) {
    nameError.value = 'Name is required.'
  } else if (Array.from(name.value.trim()).length > 80) {
    nameError.value = 'Name must be 80 characters or fewer.'
  }
  if (!email.value.trim()) {
    emailError.value = 'Email is required.'
  } else if (email.value.trim().length > 254) {
    emailError.value = 'Email must be 254 characters or fewer.'
  }
  const passwordBytes = new TextEncoder().encode(password.value).length
  if (passwordBytes < 8 || passwordBytes > 128) {
    passwordError.value = 'Password must be between 8 and 128 bytes.'
  }
  if (nameError.value || emailError.value || passwordError.value) {
    return
  }

  submitting.value = true
  try {
    await register({ name: name.value.trim(), email: email.value.trim(), password: password.value })
    await router.push({ name: 'dashboard' })
  } catch (error) {
    if (error instanceof ApiError && error.field === 'name') {
      nameError.value = error.message
    } else if (error instanceof ApiError && (error.field === 'email' || error.status === 409)) {
      emailError.value = error.message
    } else if (error instanceof ApiError && error.field === 'password') {
      passwordError.value = error.message
    } else {
      errorMessage.value = error instanceof Error ? error.message : 'Could not create your account. Please try again.'
    }
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <GuestShell header-link-label="Sign in" :header-link-to="{ name: 'login' }">
    <section class="w-full max-w-md border-y border-vigil-rule bg-vigil-surface px-5 py-8 sm:border sm:px-8" aria-labelledby="register-title">
      <h1 id="register-title" class="mt-6 text-2xl font-semibold">Create your account</h1>
      <p class="mt-2 text-sm text-vigil-muted">Start monitoring your websites and services.</p>

      <p v-if="errorMessage" class="mt-5 border-l-2 border-vigil-down bg-vigil-down-soft px-4 py-3 text-sm text-vigil-down" role="alert">
        {{ errorMessage }}
      </p>

      <form class="mt-6 grid gap-5" @submit.prevent="submit">
        <FormField
          id="register-name"
          v-model="name"
          label="Name"
          autocomplete="name"
          :max-length="80"
          :disabled="submitting"
          :error="nameError"
          required
        />
        <FormField
          id="register-email"
          v-model="email"
          label="Email"
          type="email"
          autocomplete="email"
          :max-length="254"
          :disabled="submitting"
          :error="emailError"
          required
        />
        <FormField
          id="register-password"
          v-model="password"
          label="Password"
          type="password"
          autocomplete="new-password"
          :max-length="128"
          :disabled="submitting"
          :error="passwordError"
          required
        />
        <p class="-mt-3 text-xs text-vigil-muted">Use 8 to 128 bytes.</p>
        <button
          class="min-h-11 rounded-md bg-vigil-brand px-4 py-2 font-medium text-white hover:bg-vigil-brand-hover focus-visible:outline-2 focus-visible:outline-offset-2 disabled:cursor-wait disabled:opacity-60"
          type="submit"
          :disabled="submitting"
        >
          {{ submitting ? 'Creating account…' : 'Create account' }}
        </button>
      </form>

    </section>
  </GuestShell>
</template>
