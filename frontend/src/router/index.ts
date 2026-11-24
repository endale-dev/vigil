import { createRouter, createWebHistory } from 'vue-router'
import AppShell from '../components/AppShell.vue'
import NotFoundView from '../views/NotFoundView.vue'
import { useSession } from '../composables/useSession'

declare module 'vue-router' {
  interface RouteMeta {
    title?: string
    requiresAuth?: boolean
    guestOnly?: boolean
  }
}

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: { name: 'dashboard' } },
    { path: '/login', name: 'login', component: AppShell, meta: { title: 'Sign in', guestOnly: true } },
    { path: '/register', name: 'register', component: AppShell, meta: { title: 'Create account', guestOnly: true } },
    { path: '/forgot-password', name: 'forgot-password', component: AppShell, meta: { title: 'Reset password' } },
    { path: '/reset-password', name: 'reset-password', component: AppShell, meta: { title: 'Choose a new password' } },
    { path: '/dashboard', name: 'dashboard', component: AppShell, meta: { title: 'Dashboard', requiresAuth: true } },
    { path: '/monitors/:id', name: 'monitor-detail', component: AppShell, meta: { title: 'Monitor', requiresAuth: true } },
    { path: '/settings', name: 'settings', component: AppShell, meta: { title: 'Settings', requiresAuth: true } },
    { path: '/status/:slug', name: 'public-status', component: AppShell, meta: { title: 'Public status' } },
    { path: '/:pathMatch(.*)*', name: 'not-found', component: NotFoundView },
  ],
})

router.beforeEach(async (to) => {
  const session = useSession()
  if ((to.meta.requiresAuth || to.meta.guestOnly) && !session.isReady.value) {
    await session.initialize().catch(() => null)
  }

  if (to.meta.requiresAuth && !session.isAuthenticated.value) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if (to.meta.guestOnly && session.isAuthenticated.value) {
    return { name: 'dashboard' }
  }
  return true
})

export { router }
