import { onMounted, onUnmounted, readonly } from 'vue'
import { useAsyncResource } from './useAsyncResource'

export function usePolling<T>(load: (signal: AbortSignal) => Promise<T>, intervalMs: number) {
  if (!Number.isFinite(intervalMs) || intervalMs <= 0) {
    throw new RangeError('Polling interval must be greater than zero.')
  }

  const { data, error, loading: isLoading, refresh } = useAsyncResource(load)
  let timer: ReturnType<typeof setInterval> | undefined

  onMounted(() => {
    timer = setInterval(() => {
      if (!isLoading.value) void refresh()
    }, intervalMs)
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
