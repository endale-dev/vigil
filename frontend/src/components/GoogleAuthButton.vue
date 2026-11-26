<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { getCapabilities, startGoogleLogin } from '../api/auth'

defineProps<{
  label: string
  disabled?: boolean
}>()

const enabled = ref(false)

onMounted(async () => {
  try {
    enabled.value = (await getCapabilities()).google
  } catch {
    enabled.value = false
  }
})
</script>

<template>
  <div v-if="enabled" class="google-auth">
    <button class="google-auth-button" type="button" :disabled="disabled" @click="startGoogleLogin">
      <svg class="google-auth-mark" viewBox="0 0 24 24" aria-hidden="true">
        <path fill="#4285F4" d="M21.35 12.24c0-.71-.06-1.39-.18-2.04H12v3.87h5.24a4.48 4.48 0 0 1-1.95 2.94v2.44h3.15c1.84-1.69 2.91-4.18 2.91-7.21Z" />
        <path fill="#34A853" d="M12 21.75c2.63 0 4.84-.87 6.45-2.3l-3.15-2.44c-.87.58-1.98.93-3.3.93a5.94 5.94 0 0 1-5.59-4.12H3.16v2.51A9.75 9.75 0 0 0 12 21.75Z" />
        <path fill="#FBBC05" d="M6.41 13.82a5.86 5.86 0 0 1 0-3.64V7.67H3.16a9.75 9.75 0 0 0 0 8.66l3.25-2.51Z" />
        <path fill="#EA4335" d="M12 6.06c1.43 0 2.72.49 3.73 1.47l2.8-2.8A9.36 9.36 0 0 0 12 2.25a9.75 9.75 0 0 0-8.84 5.42l3.25 2.51A5.94 5.94 0 0 1 12 6.06Z" />
      </svg>
      {{ label }}
    </button>
    <div class="auth-divider" aria-hidden="true"><span>or use email</span></div>
  </div>
</template>
