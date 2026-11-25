<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { Gauge, Globe, Settings } from '@lucide/vue'
import { useSession } from '../composables/useSession'

const { user } = useSession()
const navigation = computed(() => [
  { label: 'Overview', to: { name: 'dashboard' }, icon: Gauge },
  ...(user.value?.statusSlug
    ? [{ label: 'Status', to: { name: 'public-status', params: { slug: user.value.statusSlug } }, icon: Globe }]
    : []),
  { label: 'Settings', to: { name: 'settings' }, icon: Settings },
])
const initials = computed(() => user.value?.name
  .split(/\s+/)
  .filter(Boolean)
  .slice(0, 2)
  .map((part) => part[0]?.toUpperCase())
  .join('') || 'V')
</script>

<template>
  <div class="min-h-screen bg-vigil-paper font-vigil-sans text-vigil-ink antialiased">
    <aside class="fixed inset-y-0 left-0 z-30 hidden w-36 flex-col border-r border-vigil-rule bg-vigil-rail md:flex lg:w-56">
      <RouterLink
        class="mx-auto flex h-16 items-center rounded-sm text-lg font-semibold tracking-[-0.065em] text-vigil-ink focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-vigil-brand lg:mx-6"
        :to="{ name: 'dashboard' }"
        aria-label="Vigil overview"
      >
        vigil<span class="text-vigil-brand">.</span>
      </RouterLink>

      <nav class="mt-8 grid gap-1 px-3 lg:px-4" aria-label="Workspace navigation">
        <RouterLink
          v-for="item in navigation"
          :key="item.label"
          :to="item.to"
          :aria-label="item.label"
          :title="item.label"
          class="relative flex min-h-11 items-center gap-3 rounded-md px-3 text-sm font-medium text-vigil-muted transition-colors hover:bg-vigil-paper hover:text-vigil-ink focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-vigil-brand lg:px-3"
          active-class="!bg-vigil-brand-soft !text-vigil-brand before:absolute before:left-0 before:h-6 before:w-0.5 before:bg-vigil-brand"
        >
          <component :is="item.icon" :size="18" :stroke-width="1.8" class="shrink-0" aria-hidden="true" />
          <span>{{ item.label }}</span>
        </RouterLink>
      </nav>

      <div class="mt-auto border-t border-vigil-rule p-3 lg:p-4">
        <div class="flex items-center gap-3" :title="user?.name ?? 'Vigil user'">
          <span class="grid size-9 shrink-0 place-items-center rounded-full bg-vigil-brand-soft text-xs font-semibold text-vigil-brand" aria-hidden="true">
            {{ initials }}
          </span>
          <div class="hidden min-w-0 lg:block">
            <p class="truncate text-sm font-medium text-vigil-ink">{{ user?.name ?? 'Vigil user' }}</p>
            <p class="truncate text-xs text-vigil-muted">{{ user?.email ?? '' }}</p>
          </div>
        </div>
      </div>
    </aside>

    <main class="min-h-screen w-full md:ml-36 md:w-[calc(100%-9rem)] lg:ml-56 lg:w-[calc(100%-14rem)]">
      <div class="mx-auto min-h-screen w-full max-w-[1120px] px-5 pb-[calc(80px_+_env(safe-area-inset-bottom))] pt-7 sm:px-7 sm:pt-10 md:pb-12">
        <slot />
      </div>
    </main>

    <nav
      class="fixed inset-x-0 bottom-0 z-40 border-t border-vigil-rule bg-vigil-surface px-3 pt-1.5 md:hidden"
      aria-label="Primary navigation"
      style="height: calc(64px + env(safe-area-inset-bottom)); padding-bottom: env(safe-area-inset-bottom)"
    >
      <div class="mx-auto grid h-14 max-w-md gap-1" :style="{ gridTemplateColumns: `repeat(${navigation.length}, minmax(0, 1fr))` }">
        <RouterLink
          v-for="item in navigation"
          :key="item.label"
          :to="item.to"
          class="relative flex min-h-12 flex-col items-center justify-center gap-0.5 rounded-md px-1 text-[11px] font-medium text-vigil-muted transition-colors hover:text-vigil-ink focus-visible:outline-2 focus-visible:outline-offset-[-2px] focus-visible:outline-vigil-brand"
          active-class="!bg-vigil-brand-soft !font-semibold !text-vigil-brand before:absolute before:inset-x-5 before:top-0 before:h-0.5 before:bg-vigil-brand"
        >
          <component :is="item.icon" :size="18" :stroke-width="1.8" aria-hidden="true" />
          <span>{{ item.label }}</span>
        </RouterLink>
      </div>
    </nav>
  </div>
</template>
