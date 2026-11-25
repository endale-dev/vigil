<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ApiError } from '../api/client'
import FormField from '../components/FormField.vue'
import AuthLayout from '../components/AuthLayout.vue'
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
  <AuthLayout mode="register">
    <section aria-labelledby="register-title">
      <h1 id="register-title" class="console-title">Create your account</h1>

      <p v-if="errorMessage" class="mt-5 border-l-2 border-vigil-down bg-vigil-down-soft px-4 py-3 text-sm text-vigil-down" role="alert">
        {{ errorMessage }}
      </p>

      <form class="mt-6 grid gap-5" @submit.prevent="submit">
        <FormField
          id="register-name"
          v-model="name"
          label="Display name"
          placeholder="Your name"
          autocomplete="name"
          :max-length="80"
          :disabled="submitting"
          :error="nameError"
          required
        />
        <FormField
          id="register-email"
          v-model="email"
          label="Email address"
          placeholder="you@example.com"
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
          revealable
          autocomplete="new-password"
          hint="Use 8 to 128 bytes. Emoji and accented letters can use multiple bytes."
          :disabled="submitting"
          :error="passwordError"
          required
        />
        <button
          class="console-button primary entry-submit"
          type="submit"
          :disabled="submitting"
        >
          {{ submitting ? 'Creating account…' : 'Create account' }}
        </button>
      </form>

    </section>
  </AuthLayout>
</template>
