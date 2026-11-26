<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { ApiError } from '../api/client'
import FormField from '../components/FormField.vue'
import AuthLayout from '../components/AuthLayout.vue'
import GoogleAuthButton from '../components/GoogleAuthButton.vue'
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
const googleError = computed(() => route.query.error === 'google')
const resetSuccess = computed(() => route.query.reset === 'success')

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
  <AuthLayout mode="login">
    <section aria-labelledby="login-title">
      <h1 id="login-title" class="console-title">Sign in</h1>
      <p class="console-description">Welcome back to Vigil.</p>

      <p v-if="googleError" class="mt-5 border-l-2 border-vigil-down bg-vigil-down-soft px-4 py-3 text-sm text-vigil-down" role="alert">
        Google sign-in did not work. Please try again or use your email and password.
      </p>
      <p v-if="resetSuccess" class="mt-5 border-l-2 border-vigil-up bg-vigil-up-soft px-4 py-3 text-sm text-vigil-up" role="status">
        Your password was reset. Sign in with your new password.
      </p>
      <p v-if="errorMessage" class="mt-5 border-l-2 border-vigil-down bg-vigil-down-soft px-4 py-3 text-sm text-vigil-down" role="alert">
        {{ errorMessage }}
      </p>

      <GoogleAuthButton label="Continue with Google" :disabled="submitting" />

      <form class="entry-form" @submit.prevent="submit">
        <FormField
          id="login-email"
          v-model="email"
          label="Email address"
          placeholder="you@example.com"
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
          revealable
          autocomplete="current-password"
          :disabled="submitting"
          :error="passwordError"
          required
        >
          <template #label-action>
            <RouterLink class="entry-recovery" :to="{ name: 'forgot-password' }">Forgot password?</RouterLink>
          </template>
        </FormField>
        <button
          class="console-button primary entry-submit"
          type="submit"
          :disabled="submitting"
        >
          {{ submitting ? 'Signing in…' : 'Sign in' }}
        </button>
      </form>

    </section>
  </AuthLayout>
</template>
