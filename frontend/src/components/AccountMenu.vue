<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { ChevronUp, LogOut, Settings, UserRound } from '@lucide/vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { useSession } from '../composables/useSession'

const props = defineProps<{ variant: 'rail' | 'mobile'; compact?: boolean }>()
const route = useRoute()
const router = useRouter()
const { user, logout } = useSession()
const root = ref<HTMLElement | null>(null)
const trigger = ref<HTMLButtonElement | null>(null)
const open = ref(false)
const pending = ref(false)
const errorMessage = ref('')
const menuId = computed(() => `account-options-${props.variant}`)
const initials = computed(() => user.value?.name.trim().split(/\s+/).slice(0, 2).map(part => part[0]?.toUpperCase()).join('') || 'V')

watch(() => route.fullPath, () => { open.value = false })

function closeOnOutsidePointer(event: PointerEvent): void {
  if (open.value && event.target instanceof Node && !root.value?.contains(event.target)) {
    open.value = false
  }
}

function closeOnFocusOut(event: FocusEvent): void {
  if (event.relatedTarget instanceof Node && !root.value?.contains(event.relatedTarget)) {
    open.value = false
  }
}

onMounted(() => document.addEventListener('pointerdown', closeOnOutsidePointer))
onUnmounted(() => document.removeEventListener('pointerdown', closeOnOutsidePointer))

function closeFromKeyboard(event: KeyboardEvent): void {
  if (!open.value) return
  event.stopPropagation()
  open.value = false
  trigger.value?.focus()
}

async function signOut(): Promise<void> {
  pending.value = true
  errorMessage.value = ''
  try {
    await logout()
    await router.replace({ name: 'login' })
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : 'Could not log out.'
  } finally {
    pending.value = false
  }
}
</script>

<template>
  <div ref="root" class="account-menu" :class="`account-menu-${variant}`" @focusout="closeOnFocusOut" @keydown.esc="closeFromKeyboard">
    <button ref="trigger" type="button" class="account-trigger" :class="{ selected: route.name === 'account-settings' }"
      :aria-label="variant === 'rail' ? (user?.name ? `Profile for ${user.name}` : 'Profile') : undefined"
      :title="compact ? 'Profile' : undefined" :aria-expanded="open" :aria-controls="menuId" @click="open = !open">
      <template v-if="variant === 'rail'">
        <span class="console-avatar" aria-hidden="true">{{ initials }}</span>
        <span v-if="!compact" class="rail-user"><strong>{{ user?.name }}</strong><small>Profile</small></span>
        <ChevronUp v-if="!compact" :size="14" aria-hidden="true" />
      </template>
      <template v-else><UserRound :size="19" aria-hidden="true" />Profile</template>
    </button>
    <div v-show="open" :id="menuId" class="account-options">
      <RouterLink :to="{ name: 'account-settings' }" class="account-option" @click="open = false">
        <Settings :size="16" aria-hidden="true" /> Account settings
      </RouterLink>
      <button type="button" class="account-option" :disabled="pending" @click="signOut">
        <LogOut :size="16" aria-hidden="true" /> {{ pending ? 'Logging out…' : 'Log out' }}
      </button>
      <p v-if="errorMessage" class="px-2 py-1 text-sm text-vigil-down" role="alert">{{ errorMessage }}</p>
    </div>
  </div>
</template>
