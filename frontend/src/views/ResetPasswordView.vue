<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { ApiError } from '../api/client'
import { resetPassword } from '../api/auth'
import FormField from '../components/FormField.vue'

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
  <main class="flex min-h-screen items-center justify-center bg-slate-950 px-4 py-10 text-slate-100">
    <section class="w-full max-w-md rounded-xl border border-slate-800 bg-slate-900 p-6 shadow-xl sm:p-8" aria-labelledby="reset-title">
      <RouterLink class="text-lg font-semibold focus-visible:outline-2 focus-visible:outline-offset-4" :to="{ name: 'home' }">
        Vigil
      </RouterLink>
      <h1 id="reset-title" class="mt-6 text-2xl font-semibold">Choose a new password</h1>

      <p v-if="!token" class="mt-4 text-sm text-red-200" role="alert">
        This reset link is missing its token. Request a new link to continue.
      </p>
      <p v-if="errorMessage" class="mt-4 rounded-md border border-red-800 bg-red-950/60 p-3 text-sm text-red-200" role="alert">
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
        <p class="-mt-3 text-xs text-slate-400">Use 8 to 128 bytes.</p>
        <button
          class="min-h-11 rounded-md bg-emerald-700 px-4 py-2 font-medium text-white hover:bg-emerald-600 focus-visible:outline-2 focus-visible:outline-offset-2 disabled:cursor-wait disabled:opacity-60"
          type="submit"
          :disabled="submitting"
        >
          {{ submitting ? 'Saving…' : 'Reset password' }}
        </button>
      </form>

      <RouterLink class="mt-6 inline-block text-sm text-emerald-300 underline-offset-4 hover:underline focus-visible:outline-2 focus-visible:outline-offset-4" :to="{ name: 'forgot-password' }">
        Request a new reset link
      </RouterLink>
    </section>
  </main>
</template>
