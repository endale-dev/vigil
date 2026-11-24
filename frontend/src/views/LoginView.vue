<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { ApiError } from '../api/client'
import { getCapabilities, startGoogleLogin } from '../api/auth'
import FormField from '../components/FormField.vue'
import { useSession } from '../composables/useSession'

const route = useRoute()
const router = useRouter()
const { login } = useSession()
const email = ref('')
const password = ref('')
const emailError = ref('')
const passwordError = ref('')
const errorMessage = ref('')
const submitting = ref(false)
const googleEnabled = ref(false)
const googleError = computed(() => route.query.error === 'google')

onMounted(async () => {
  try {
    googleEnabled.value = (await getCapabilities()).google
  } catch {
    googleEnabled.value = false
  }
})

async function submit(): Promise<void> {
  emailError.value = ''
  passwordError.value = ''
  errorMessage.value = ''
  if (!email.value.trim()) {
    emailError.value = 'Email is required.'
  }
  if (!password.value) {
    passwordError.value = 'Password is required.'
  }
  if (emailError.value || passwordError.value) {
    return
  }

  submitting.value = true
  try {
    await login({ email: email.value, password: password.value })
    const redirect = route.query.redirect
    const destination = typeof redirect === 'string' && redirect.startsWith('/') && !redirect.startsWith('//')
      ? redirect
      : { name: 'dashboard' }
    await router.push(destination)
  } catch (error) {
    if (error instanceof ApiError && error.field === 'email') {
      emailError.value = error.message
    } else if (error instanceof ApiError && error.field === 'password') {
      passwordError.value = error.message
    } else {
      errorMessage.value = error instanceof Error ? error.message : 'Could not sign in. Please try again.'
    }
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <main class="flex min-h-screen items-center justify-center bg-slate-950 px-4 py-10 text-slate-100">
    <section class="w-full max-w-md rounded-xl border border-slate-800 bg-slate-900 p-6 shadow-xl sm:p-8" aria-labelledby="login-title">
      <RouterLink class="text-lg font-semibold focus-visible:outline-2 focus-visible:outline-offset-4" to="/login">
        Vigil
      </RouterLink>
      <h1 id="login-title" class="mt-6 text-2xl font-semibold">Sign in</h1>
      <p class="mt-2 text-sm text-slate-300">Keep an eye on the services that matter to you.</p>

      <p v-if="googleError" class="mt-5 rounded-md border border-red-800 bg-red-950/60 p-3 text-sm text-red-200" role="alert">
        Google sign-in did not work. Please try again or use your email and password.
      </p>
      <p v-if="errorMessage" class="mt-5 rounded-md border border-red-800 bg-red-950/60 p-3 text-sm text-red-200" role="alert">
        {{ errorMessage }}
      </p>

      <form class="mt-6 grid gap-5" @submit.prevent="submit">
        <FormField
          id="login-email"
          v-model="email"
          label="Email"
          type="email"
          autocomplete="email"
          :disabled="submitting"
          :error="emailError"
          required
        />
        <FormField
          id="login-password"
          v-model="password"
          label="Password"
          type="password"
          autocomplete="current-password"
          :disabled="submitting"
          :error="passwordError"
          required
        />
        <div class="flex justify-end text-sm">
          <RouterLink class="text-emerald-300 underline-offset-4 hover:underline focus-visible:outline-2 focus-visible:outline-offset-4" :to="{ name: 'forgot-password' }">
            Forgot password?
          </RouterLink>
        </div>
        <button
          class="min-h-11 rounded-md bg-emerald-700 px-4 py-2 font-medium text-white hover:bg-emerald-600 focus-visible:outline-2 focus-visible:outline-offset-2 disabled:cursor-wait disabled:opacity-60"
          type="submit"
          :disabled="submitting"
        >
          {{ submitting ? 'Signing in…' : 'Sign in' }}
        </button>
      </form>

      <template v-if="googleEnabled">
        <div class="my-5 flex items-center gap-3 text-xs text-slate-500" aria-hidden="true">
          <span class="h-px flex-1 bg-slate-700" />
          <span>OR</span>
          <span class="h-px flex-1 bg-slate-700" />
        </div>
        <button
          class="min-h-11 w-full rounded-md border border-slate-600 px-4 py-2 font-medium hover:bg-slate-800 focus-visible:outline-2 focus-visible:outline-offset-2"
          type="button"
          @click="startGoogleLogin"
        >
          Continue with Google
        </button>
      </template>

      <p class="mt-6 text-center text-sm text-slate-300">
        New to Vigil?
        <RouterLink class="text-emerald-300 underline-offset-4 hover:underline focus-visible:outline-2 focus-visible:outline-offset-4" :to="{ name: 'register' }">
          Create an account
        </RouterLink>
      </p>
    </section>
  </main>
</template>
