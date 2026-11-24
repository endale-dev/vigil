<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { ApiError } from '../api/client'
import FormField from '../components/FormField.vue'
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
  <main class="flex min-h-screen items-center justify-center bg-slate-950 px-4 py-10 text-slate-100">
    <section class="w-full max-w-md rounded-xl border border-slate-800 bg-slate-900 p-6 shadow-xl sm:p-8" aria-labelledby="register-title">
      <RouterLink class="text-lg font-semibold focus-visible:outline-2 focus-visible:outline-offset-4" to="/register">
        Vigil
      </RouterLink>
      <h1 id="register-title" class="mt-6 text-2xl font-semibold">Create your account</h1>
      <p class="mt-2 text-sm text-slate-300">Start monitoring your websites and services.</p>

      <p v-if="errorMessage" class="mt-5 rounded-md border border-red-800 bg-red-950/60 p-3 text-sm text-red-200" role="alert">
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
        <p class="-mt-3 text-xs text-slate-400">Use 8 to 128 bytes.</p>
        <button
          class="min-h-11 rounded-md bg-emerald-700 px-4 py-2 font-medium text-white hover:bg-emerald-600 focus-visible:outline-2 focus-visible:outline-offset-2 disabled:cursor-wait disabled:opacity-60"
          type="submit"
          :disabled="submitting"
        >
          {{ submitting ? 'Creating account…' : 'Create account' }}
        </button>
      </form>

      <p class="mt-6 text-center text-sm text-slate-300">
        Already have an account?
        <RouterLink class="text-emerald-300 underline-offset-4 hover:underline focus-visible:outline-2 focus-visible:outline-offset-4" :to="{ name: 'login' }">
          Sign in
        </RouterLink>
      </p>
    </section>
  </main>
</template>
