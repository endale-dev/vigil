<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { Activity, Globe, LayoutDashboard, Menu, Settings, X } from '@lucide/vue'
import { useSession } from '../composables/useSession'

defineProps<{
  title: string
  description?: string
}>()

const route = useRoute()
const { user } = useSession()
const mobileNavOpen = ref(false)
const navigation = computed(() => [
  { label: 'Overview', to: { name: 'dashboard' }, icon: LayoutDashboard },
  ...(user.value?.statusSlug
    ? [{ label: 'Public status', to: { name: 'public-status', params: { slug: user.value.statusSlug } }, icon: Globe }]
    : []),
  { label: 'Settings', to: { name: 'settings' }, icon: Settings },
])
const initials = computed(() => user.value?.name
  .split(/\s+/)
  .filter(Boolean)
  .slice(0, 2)
  .map((part) => part[0]?.toUpperCase())
  .join('') || 'V')

watch(() => route.fullPath, () => {
  mobileNavOpen.value = false
})
</script>

<template>
  <div class="min-h-screen bg-vigil-canvas text-vigil-ink antialiased">
    <button
      v-if="mobileNavOpen"
      class="fixed inset-0 z-40 bg-slate-950/35 lg:hidden"
      type="button"
      aria-label="Close navigation"
      @click="mobileNavOpen = false"
    />

    <aside
      id="workspace-navigation"
      class="fixed inset-y-0 left-0 z-50 flex w-[17rem] flex-col bg-vigil-sidebar text-white shadow-xl shadow-slate-950/10 transition-transform duration-200 lg:translate-x-0"
      :class="mobileNavOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0'"
    >
      <div class="flex h-[4.5rem] items-center justify-between border-b border-white/10 px-6">
        <RouterLink class="inline-flex items-center gap-3 rounded-md font-semibold tracking-tight focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-white" :to="{ name: 'dashboard' }">
          <span class="grid size-8 place-items-center rounded-lg bg-white text-vigil-sidebar">
            <Activity :size="18" :stroke-width="2.25" aria-hidden="true" />
          </span>
          <span class="text-lg">Vigil</span>
        </RouterLink>
        <button
          class="grid size-9 place-items-center rounded-lg text-slate-300 hover:bg-white/10 hover:text-white focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white lg:hidden"
          type="button"
          aria-label="Close navigation"
          @click="mobileNavOpen = false"
        >
          <X :size="19" aria-hidden="true" />
        </button>
      </div>

      <div class="px-5 pt-7">
        <p class="px-3 text-[11px] font-semibold uppercase tracking-[0.16em] text-slate-400">Workspace</p>
        <nav class="mt-3 grid gap-1" aria-label="Workspace navigation">
          <RouterLink
            v-for="item in navigation"
            :key="item.label"
            :to="item.to"
            class="group relative flex min-h-11 items-center gap-3 rounded-lg px-3 text-sm font-medium text-slate-300 transition-colors hover:bg-white/10 hover:text-white focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white"
            active-class="!bg-white/12 !text-white before:absolute before:left-0 before:h-6 before:w-[3px] before:rounded-r before:bg-[#9fb4ff]"
          >
            <component :is="item.icon" :size="18" :stroke-width="1.8" class="shrink-0 text-slate-400 group-hover:text-white" aria-hidden="true" />
            <span>{{ item.label }}</span>
          </RouterLink>
        </nav>
      </div>

      <div class="mt-auto border-t border-white/10 p-4">
        <div class="flex items-center gap-3 rounded-xl bg-white/5 p-3">
          <span class="grid size-9 shrink-0 place-items-center rounded-full bg-[#dce5ff] text-xs font-semibold text-[#263c83]" aria-hidden="true">
            {{ initials }}
          </span>
          <div class="min-w-0">
            <p class="truncate text-sm font-medium text-white">{{ user?.name ?? 'Vigil user' }}</p>
            <p class="truncate text-xs text-slate-400">{{ user?.email ?? '' }}</p>
          </div>
        </div>
      </div>
    </aside>

    <div class="min-h-screen lg:pl-[17rem]">
      <header class="sticky top-0 z-30 border-b border-vigil-line bg-white/95 backdrop-blur-sm">
        <div class="mx-auto flex h-[4.5rem] max-w-[1440px] items-center justify-between gap-4 px-4 sm:px-7 lg:px-10">
          <div class="flex min-w-0 items-center gap-3">
            <button
              class="grid size-10 shrink-0 place-items-center rounded-lg border border-vigil-line text-vigil-ink hover:bg-vigil-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-vigil-brand lg:hidden"
              type="button"
              aria-label="Open navigation"
              aria-controls="workspace-navigation"
              :aria-expanded="mobileNavOpen"
              @click="mobileNavOpen = true"
            >
              <Menu :size="19" aria-hidden="true" />
            </button>
            <div class="min-w-0">
              <p class="truncate text-sm font-semibold">{{ title }}</p>
              <p v-if="description" class="hidden truncate text-xs text-vigil-muted sm:block">{{ description }}</p>
            </div>
          </div>
          <slot name="header-action" />
        </div>
      </header>

      <main class="mx-auto w-full max-w-[1440px] px-4 py-7 sm:px-7 sm:py-9 lg:px-10 lg:py-10">
        <slot />
      </main>
    </div>
  </div>
</template>
