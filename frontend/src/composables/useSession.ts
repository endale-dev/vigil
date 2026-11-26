import { computed, readonly, ref } from 'vue'
import { ApiError, setUnauthorizedHandler } from '../api/client'
import {
  getCurrentUser,
  login as loginRequest,
  logout as logoutRequest,
  register as registerRequest,
} from '../api/auth'
import type { LoginRequest, RegisterRequest, UserDTO } from '../api/types'

const currentUser = ref<UserDTO | null>(null)
const sessionReady = ref(false)
let sessionVersion = 0
let refreshRequest: Promise<UserDTO | null> | null = null

setUnauthorizedHandler(() => {
  sessionVersion++
  currentUser.value = null
  sessionReady.value = true
})

async function refresh(): Promise<UserDTO | null> {
  if (refreshRequest) {
    return refreshRequest
  }

  const version = ++sessionVersion
  const request = getCurrentUser()
    .then((user) => {
      if (version !== sessionVersion) {
        return currentUser.value
      }
      currentUser.value = user
      return user
    })
    .catch((error: unknown) => {
      if (version !== sessionVersion) {
        return currentUser.value
      }
      if (error instanceof ApiError && error.status === 401) {
        currentUser.value = null
        return null
      }
      throw error
    })
    .finally(() => {
      if (refreshRequest === request) {
        refreshRequest = null
      }
      if (version === sessionVersion) {
        sessionReady.value = true
      }
    })
  refreshRequest = request
  return request
}

function initialize(): Promise<UserDTO | null> {
  return sessionReady.value ? Promise.resolve(currentUser.value) : refresh()
}

async function login(input: LoginRequest): Promise<UserDTO> {
  const version = ++sessionVersion
  const user = await loginRequest(input)
  if (version === sessionVersion) {
    currentUser.value = user
    sessionReady.value = true
  }
  return user
}

async function register(input: RegisterRequest): Promise<UserDTO> {
  const version = ++sessionVersion
  const user = await registerRequest(input)
  if (version === sessionVersion) {
    currentUser.value = user
    sessionReady.value = true
  }
  return user
}

async function logout(): Promise<void> {
  const version = ++sessionVersion
  await logoutRequest()
  if (version === sessionVersion) {
    currentUser.value = null
    sessionReady.value = true
  }
}

function applyNotifications(updated: UserDTO): void {
  if (currentUser.value?.id === updated.id) {
    currentUser.value = { ...currentUser.value, notify: updated.notify }
  }
}

export function useSession() {
  return {
    user: readonly(currentUser),
    isAuthenticated: computed(() => currentUser.value !== null),
    isReady: readonly(sessionReady),
    initialize,
    refresh,
    applyNotifications,
    login,
    register,
    logout,
  }
}
