import { onMounted, onUnmounted, readonly, ref, shallowRef } from 'vue'

export function usePolling<T>(load: () => Promise<T>, intervalMs: number) {
  if (!Number.isFinite(intervalMs) || intervalMs <= 0) {
    throw new RangeError('Polling interval must be greater than zero.')
  }

  const data = shallowRef<T | null>(null)
  const error = shallowRef<unknown>(null)
  const isLoading = ref(false)
  let timer: ReturnType<typeof setInterval> | undefined
  let requestInFlight = false

  async function refresh(): Promise<void> {
    if (requestInFlight) {
      return
    }

    requestInFlight = true
    isLoading.value = true
    try {
      data.value = await load()
      error.value = null
    } catch (caught: unknown) {
      error.value = caught
    } finally {
      requestInFlight = false
      isLoading.value = false
    }
  }

  onMounted(() => {
    void refresh()
    timer = setInterval(() => void refresh(), intervalMs)
  })

  onUnmounted(() => {
    if (timer !== undefined) {
      clearInterval(timer)
      timer = undefined
    }
  })

  return {
    data: readonly(data),
    error: readonly(error),
    isLoading: readonly(isLoading),
    refresh,
  }
}
