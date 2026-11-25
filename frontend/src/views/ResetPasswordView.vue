<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ApiError } from '../api/client'
import { resetPassword } from '../api/auth'
import FormField from '../components/FormField.vue'
import GuestShell from '../components/GuestShell.vue'

const route = useRoute()
const router = useRouter()
const token = computed(() => typeof route.query.token === 'string' ? route.query.token : '')
const newPassword = ref('')
const passwordError = ref('')
const errorMessage = ref('')
const submitting = ref(false)

async function submit(): Promise<void> {
  passwordError.value = ''
  errorMessage.value = ''
  const passwordBytes = new TextEncoder().encode(newPassword.value).length
  if (passwordBytes < 8 || passwordBytes > 128) {
    passwordError.value = 'Password must be between 8 and 128 bytes.'
    return
  }

  submitting.value = true
  try {
    await resetPassword(token.value, newPassword.value)
    await router.replace({ name: 'login', query: { reset: 'success' } })
  } catch (error) {
    if (error instanceof ApiError && error.status === 400) {
      errorMessage.value = 'This reset link is invalid or has expired. Request a new one to continue.'
    } else {
      errorMessage.value = 'We could not reset your password right now. Please try again.'
    }
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <GuestShell header-link-label="Back to sign in" :header-link-to="{ name: 'login' }">
    <section class="w-full max-w-md border-y border-vigil-rule bg-vigil-surface px-5 py-8 sm:border sm:px-8" aria-labelledby="reset-title">
      <h1 id="reset-title" class="mt-6 text-2xl font-semibold">Choose a new password</h1>

      <p v-if="!token" class="mt-4 text-sm text-vigil-down" role="alert">
        This reset link is missing its token. Request a new link to continue.
      </p>
      <p v-if="errorMessage" class="mt-4 border-l-2 border-vigil-down bg-vigil-down-soft px-4 py-3 text-sm text-vigil-down" role="alert">
        {{ errorMessage }}
      </p>

      <form v-if="token" class="mt-6 grid gap-5" @submit.prevent="submit">
        <FormField
          id="reset-password"
          v-model="newPassword"
          label="New password"
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
          {{ submitting ? 'Saving…' : 'Reset password' }}
        </button>
      </form>

    </section>
  </GuestShell>
</template>
