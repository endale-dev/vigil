<script setup lang="ts">
import '../console.css'
import { RouterLink } from 'vue-router'
import VigilBrand from './VigilBrand.vue'

defineProps<{ mode: 'login' | 'register' }>()
</script>

<template>
  <main class="account-entry console-theme">
    <div class="entry-content">
      <RouterLink class="entry-brand" :to="{ name: 'home' }" aria-label="Vigil home"><VigilBrand /></RouterLink>
      <slot />
      <p class="entry-switch">
        {{ mode === 'login' ? 'New to Vigil?' : 'Already have an account?' }}
        <RouterLink :to="{ name: mode === 'login' ? 'register' : 'login' }">
          {{ mode === 'login' ? 'Create an account' : 'Sign in' }}
        </RouterLink>
      </p>
    </div>
  </main>
</template>
