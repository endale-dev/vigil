<script setup lang="ts">
import { RouterLink } from 'vue-router'
</script>

<template>
  <main class="mx-auto flex min-h-screen max-w-xl flex-col items-start justify-center gap-4 px-6 text-slate-100">
    <p class="text-sm font-medium uppercase tracking-wide text-slate-400">404</p>
    <h1 class="text-3xl font-semibold">Page not found</h1>
    <p class="text-slate-300">That page does not exist or may have moved.</p>
    <RouterLink class="underline underline-offset-4 focus-visible:outline-2 focus-visible:outline-offset-4" to="/dashboard">
      Go to dashboard
    </RouterLink>
  </main>
</template>
