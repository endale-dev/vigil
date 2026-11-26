import { onMounted, onScopeDispose, ref, shallowRef } from 'vue'

export function useAsyncResource<T>(load: (signal: AbortSignal) => Promise<T>) {
  const data = shallowRef<T | null>(null)
  const error = shallowRef<Error | null>(null)
  const loading = ref(true)
  let controller: AbortController | undefined

  async function refresh() {
    controller?.abort()
    const request = new AbortController()
    controller = request
    loading.value = true
    error.value = null
    try {
      const result = await load(request.signal)
      if (!request.signal.aborted) data.value = result
    } catch (cause) {
      if (!request.signal.aborted) {
        error.value = cause instanceof Error ? cause : new Error('Could not load this information.')
      }
    } finally {
      if (!request.signal.aborted) loading.value = false
    }
  }

  onMounted(refresh)
  onScopeDispose(() => controller?.abort())
  return { data, error, loading, refresh }
}
