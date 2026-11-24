<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { useSession } from '../composables/useSession'

const route = useRoute()
const { isAuthenticated, user } = useSession()
const title = computed(() => route.meta.title ?? 'Vigil')
</script>

<template>
  <div class="min-h-screen bg-slate-950 text-slate-100">
    <header class="border-b border-slate-800 px-6 py-4">
      <nav class="mx-auto flex max-w-6xl items-center justify-between" aria-label="Main navigation">
        <RouterLink class="text-lg font-semibold focus-visible:outline-2 focus-visible:outline-offset-4" to="/dashboard">
          Vigil
        </RouterLink>
        <div class="flex items-center gap-5 text-sm">
          <template v-if="isAuthenticated">
            <RouterLink class="focus-visible:outline-2 focus-visible:outline-offset-4" :to="{ name: 'dashboard' }">
              Dashboard
            </RouterLink>
            <RouterLink class="focus-visible:outline-2 focus-visible:outline-offset-4" :to="{ name: 'settings' }">
              Settings
            </RouterLink>
            <RouterLink
              v-if="user?.statusSlug"
              class="focus-visible:outline-2 focus-visible:outline-offset-4"
              :to="{ name: 'public-status', params: { slug: user.statusSlug } }"
            >
              Public status
            </RouterLink>
          </template>
          <template v-else>
            <RouterLink class="focus-visible:outline-2 focus-visible:outline-offset-4" :to="{ name: 'login' }">
              Sign in
            </RouterLink>
            <RouterLink class="focus-visible:outline-2 focus-visible:outline-offset-4" :to="{ name: 'register' }">
              Register
            </RouterLink>
          </template>
        </div>
      </nav>
    </header>
    <main class="mx-auto max-w-6xl px-6 py-8">
      <h1 class="text-2xl font-semibold">{{ title }}</h1>
    </main>
  </div>
</template>
