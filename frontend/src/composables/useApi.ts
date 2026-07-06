import { ref, type Ref } from 'vue'
import type { AxiosError } from 'axios'

interface ApiState<T> {
  data: Ref<T | null>
  loading: Ref<boolean>
  error: Ref<string | null>
  execute: () => Promise<T | null>
}

/**
 * Wrapper para chamadas de API com estado loading/error automático.
 *
 * @example
 * const { data, loading, error, execute } = useApi(() => gruposApi.list())
 * onMounted(execute)
 */
export function useApi<T>(fn: () => Promise<{ data: { data: T } }>): ApiState<T> {
  const data    = ref<T | null>(null) as Ref<T | null>
  const loading = ref(false)
  const error   = ref<string | null>(null)

  async function execute(): Promise<T | null> {
    loading.value = true
    error.value   = null
    try {
      const res  = await fn()
      data.value = res.data.data
      return data.value
    } catch (e) {
      const err  = e as AxiosError<{ message: string }>
      error.value = err.response?.data?.message ?? err.message ?? 'Erro inesperado'
      return null
    } finally {
      loading.value = false
    }
  }

  return { data, loading, error, execute }
}

/** Versão para listas paginadas */
export function usePagedApi<T>(fn: (page: number, perPage: number) => Promise<{
  data: { data: T[]; meta: { page: number; per_page: number; total: number } }
}>) {
  const items   = ref<T[]>([]) as Ref<T[]>
  const loading = ref(false)
  const error   = ref<string | null>(null)
  const page    = ref(1)
  const perPage = ref(20)
  const total   = ref(0)

  async function fetch() {
    loading.value = true
    error.value   = null
    try {
      const res  = await fn(page.value, perPage.value)
      items.value = res.data.data
      total.value = res.data.meta.total
    } catch (e) {
      const err   = e as AxiosError<{ message: string }>
      error.value = err.response?.data?.message ?? 'Erro ao carregar'
    } finally {
      loading.value = false
    }
  }

  function goTo(p: number) {
    page.value = p
    fetch()
  }

  return { items, loading, error, page, perPage, total, fetch, goTo }
}
