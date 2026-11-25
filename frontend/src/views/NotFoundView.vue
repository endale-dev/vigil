<script setup lang="ts">
import { RouterLink } from 'vue-router'
import GuestShell from '../components/GuestShell.vue'
</script>

<template>
  <GuestShell>
    <section class="w-full max-w-xl border-y border-vigil-rule bg-vigil-surface px-5 py-8 sm:border sm:px-8">
      <p class="text-sm font-medium text-vigil-brand">404</p>
      <h1 class="mt-2 text-3xl font-semibold">Page not found</h1>
      <p class="mt-3 text-vigil-muted">That page does not exist or may have moved.</p>
      <RouterLink class="mt-6 inline-block text-sm font-medium text-vigil-brand underline-offset-4 hover:underline focus-visible:outline-2 focus-visible:outline-offset-4" to="/dashboard">Go to dashboard</RouterLink>
    </section>
  </GuestShell>
</template>
