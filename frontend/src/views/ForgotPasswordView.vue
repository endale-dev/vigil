<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink } from 'vue-router'
import { ApiError } from '../api/client'
import { requestPasswordReset } from '../api/auth'
import FormField from '../components/FormField.vue'

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
  <main class="flex min-h-screen items-center justify-center bg-slate-950 px-4 py-10 text-slate-100">
    <section class="w-full max-w-md rounded-xl border border-slate-800 bg-slate-900 p-6 shadow-xl sm:p-8" aria-labelledby="forgot-title">
      <RouterLink class="text-lg font-semibold focus-visible:outline-2 focus-visible:outline-offset-4" :to="{ name: 'home' }">
        Vigil
      </RouterLink>
      <h1 id="forgot-title" class="mt-6 text-2xl font-semibold">Reset your password</h1>
      <p class="mt-2 text-sm text-slate-300">Enter your account email and we’ll send a reset link if an account matches.</p>

      <p v-if="submitted" class="mt-5 rounded-md border border-emerald-800 bg-emerald-950/60 p-3 text-sm text-emerald-200" role="status">
        If an account exists for that email, we sent a password reset link.
      </p>
      <p v-if="errorMessage" class="mt-5 rounded-md border border-red-800 bg-red-950/60 p-3 text-sm text-red-200" role="alert">
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
          class="min-h-11 rounded-md bg-emerald-700 px-4 py-2 font-medium text-white hover:bg-emerald-600 focus-visible:outline-2 focus-visible:outline-offset-2 disabled:cursor-wait disabled:opacity-60"
          type="submit"
          :disabled="submitting"
        >
          {{ submitting ? 'Sending…' : 'Send reset link' }}
        </button>
      </form>

      <RouterLink class="mt-6 inline-block text-sm text-emerald-300 underline-offset-4 hover:underline focus-visible:outline-2 focus-visible:outline-offset-4" :to="{ name: 'login' }">
        Back to sign in
      </RouterLink>
    </section>
  </main>
</template>
