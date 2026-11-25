<script setup lang="ts">
import type { RouteLocationRaw } from 'vue-router'
import { RouterLink } from 'vue-router'

defineProps<{
  headerLinkLabel?: string
  headerLinkTo?: RouteLocationRaw
}>()
</script>

<template>
  <div class="min-h-screen bg-vigil-paper font-vigil-sans text-vigil-ink antialiased">
    <header class="border-b border-vigil-rule">
      <div class="mx-auto flex h-16 max-w-[1180px] items-center justify-between px-5 sm:px-7">
        <RouterLink
          class="rounded-sm text-[1.35rem] font-semibold tracking-[-0.065em] focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-vigil-brand"
          :to="{ name: 'home' }"
        >
          vigil<span class="text-vigil-brand">.</span>
        </RouterLink>
        <RouterLink
          v-if="headerLinkLabel && headerLinkTo"
          class="rounded-sm text-sm font-medium text-vigil-brand underline-offset-4 hover:underline focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-vigil-brand"
          :to="headerLinkTo"
        >
          {{ headerLinkLabel }}
        </RouterLink>
      </div>
    </header>
    <main class="mx-auto flex min-h-[calc(100vh-4rem)] w-full max-w-[1180px] items-center justify-center px-5 py-10 sm:px-7">
      <slot />
    </main>
  </div>
</template>
