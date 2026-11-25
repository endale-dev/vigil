<script setup lang="ts">
import { ref } from 'vue'
import { ApiError } from '../api/client'
import { requestPasswordReset } from '../api/auth'
import FormField from '../components/FormField.vue'
import GuestShell from '../components/GuestShell.vue'

const email = ref('')
const emailError = ref('')
const errorMessage = ref('')
const submitted = ref(false)
const submitting = ref(false)

async function submit(): Promise<void> {
  emailError.value = ''
  errorMessage.value = ''
  submitted.value = false
  submitting.value = true

  try {
    await requestPasswordReset(email.value.trim())
    submitted.value = true
  } catch (error) {
    if (error instanceof ApiError && error.field === 'email') {
      emailError.value = error.message
    } else {
      errorMessage.value = 'We could not process the request right now. Please try again.'
    }
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <GuestShell header-link-label="Back to sign in" :header-link-to="{ name: 'login' }">
    <section class="w-full max-w-md border-y border-vigil-rule bg-vigil-surface px-5 py-8 sm:border sm:px-8" aria-labelledby="forgot-title">
      <h1 id="forgot-title" class="mt-6 text-2xl font-semibold">Reset your password</h1>
      <p class="mt-2 text-sm text-vigil-muted">Enter your account email and we’ll send a reset link if an account matches.</p>

      <p v-if="submitted" class="mt-5 border-l-2 border-vigil-up bg-vigil-up-soft px-4 py-3 text-sm text-vigil-up" role="status">
        If an account exists for that email, we sent a password reset link.
      </p>
      <p v-if="errorMessage" class="mt-5 border-l-2 border-vigil-down bg-vigil-down-soft px-4 py-3 text-sm text-vigil-down" role="alert">
        {{ errorMessage }}
      </p>

      <form class="mt-6 grid gap-5" @submit.prevent="submit">
        <FormField
          id="forgot-email"
          v-model="email"
          label="Email"
          type="email"
          autocomplete="email"
          :max-length="254"
          :disabled="submitting"
          :error="emailError"
          required
        />
        <button
          class="min-h-11 rounded-md bg-vigil-brand px-4 py-2 font-medium text-white hover:bg-vigil-brand-hover focus-visible:outline-2 focus-visible:outline-offset-2 disabled:cursor-wait disabled:opacity-60"
          type="submit"
          :disabled="submitting"
        >
          {{ submitting ? 'Sending…' : 'Send reset link' }}
        </button>
      </form>

    </section>
  </GuestShell>
</template>
