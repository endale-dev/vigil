<script setup lang="ts">
import '../console.css'
import { ref } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { Bell, Globe, LayoutDashboard, PanelLeftClose, PanelLeftOpen } from '@lucide/vue'
import AccountMenu from './AccountMenu.vue'
import VigilBrand from './VigilBrand.vue'

const route = useRoute()
const storageKey = 'vigil.sidebar.collapsed'
const collapsed = ref(localStorage.getItem(storageKey) === 'true')
const navigation = [
  { label: 'Overview', to: { name: 'dashboard' }, icon: LayoutDashboard },
  { label: 'Alerts', to: { name: 'alerts' }, icon: Bell },
  { label: 'Status page', mobileLabel: 'Status page', to: { name: 'status-manager' }, icon: Globe },
]

function toggleSidebar(): void {
  collapsed.value = !collapsed.value
  localStorage.setItem(storageKey, String(collapsed.value))
}
</script>

<template>
  <div class="workspace console-theme" :class="{ 'rail-collapsed': collapsed }">
    <a class="skip-content" href="#workspace-main">Skip to content</a>
    <aside id="workspace-sidebar" class="workspace-rail">
      <div class="rail-head">
        <RouterLink class="rail-brand" :to="{ name: 'dashboard' }" aria-label="Vigil overview">
          <VigilBrand :compact="collapsed" />
        </RouterLink>
        <button class="icon-control" type="button" :aria-label="collapsed ? 'Expand sidebar' : 'Collapse sidebar'"
          :title="collapsed ? 'Expand sidebar' : 'Collapse sidebar'" :aria-expanded="!collapsed" aria-controls="workspace-sidebar" @click="toggleSidebar">
          <component :is="collapsed ? PanelLeftOpen : PanelLeftClose" :size="16" aria-hidden="true" />
        </button>
      </div>
      <nav aria-label="Workspace navigation" class="rail-nav">
        <RouterLink v-for="item in navigation" :key="item.label" :to="item.to" :title="collapsed ? item.label : undefined"
          :aria-label="item.label" class="rail-link" active-class="selected" :class="{ selected: item.to.name === 'dashboard' && route.name === 'monitor-detail' }">
          <component :is="item.icon" :size="18" aria-hidden="true" /><span v-if="!collapsed">{{ item.label }}</span>
        </RouterLink>
      </nav>
      <AccountMenu variant="rail" :compact="collapsed" />
    </aside>
    <div class="workspace-body">
      <main id="workspace-main" class="workspace-main" tabindex="-1"><slot /></main>
    </div>
    <nav class="mobile-nav" aria-label="Primary navigation">
      <RouterLink v-for="item in navigation" :key="item.label" :to="item.to" active-class="selected"
        :class="{ selected: item.to.name === 'dashboard' && route.name === 'monitor-detail' }">
        <component :is="item.icon" :size="19" aria-hidden="true" />{{ item.mobileLabel ?? item.label }}
      </RouterLink>
      <AccountMenu variant="mobile" />
    </nav>
  </div>
</template>
