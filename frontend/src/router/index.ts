import { createRouter, createWebHistory } from 'vue-router'
import AppShell from '../components/AppShell.vue'
import NotFoundView from '../views/NotFoundView.vue'
import HomeView from '../views/HomeView.vue'
import DashboardView from '../views/DashboardView.vue'
import ForgotPasswordView from '../views/ForgotPasswordView.vue'
import LoginView from '../views/LoginView.vue'
import RegisterView from '../views/RegisterView.vue'
import ResetPasswordView from '../views/ResetPasswordView.vue'
import { useSession } from '../composables/useSession'

declare module 'vue-router' {
  interface RouteMeta {
    title?: string
    requiresAuth?: boolean
    guestOnly?: boolean
    redirectAuthenticated?: boolean
  }
}

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'home', component: HomeView, meta: { title: 'Vigil', redirectAuthenticated: true } },
    { path: '/login', name: 'login', component: LoginView, meta: { title: 'Sign in', guestOnly: true } },
    { path: '/register', name: 'register', component: RegisterView, meta: { title: 'Create account', guestOnly: true } },
    { path: '/forgot-password', name: 'forgot-password', component: ForgotPasswordView, meta: { title: 'Reset password' } },
    { path: '/reset-password', name: 'reset-password', component: ResetPasswordView, meta: { title: 'Choose a new password' } },
    { path: '/dashboard', name: 'dashboard', component: DashboardView, meta: { title: 'Dashboard', requiresAuth: true } },
    { path: '/monitors/:id', name: 'monitor-detail', component: AppShell, meta: { title: 'Monitor', requiresAuth: true } },
    { path: '/settings', name: 'settings', component: AppShell, meta: { title: 'Settings', requiresAuth: true } },
    { path: '/status/:slug', name: 'public-status', component: AppShell, meta: { title: 'Public status' } },
    { path: '/:pathMatch(.*)*', name: 'not-found', component: NotFoundView },
  ],
})

router.beforeEach(async (to) => {
  const session = useSession()
  if ((to.meta.requiresAuth || to.meta.guestOnly || to.meta.redirectAuthenticated) && !session.isReady.value) {
    await session.initialize().catch(() => null)
  }

  if (to.meta.requiresAuth && !session.isAuthenticated.value) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if (to.meta.guestOnly && session.isAuthenticated.value) {
    return { name: 'dashboard' }
  }
  if (to.meta.redirectAuthenticated && session.isAuthenticated.value) {
    return { name: 'dashboard' }
  }
  return true
})

export { router }
