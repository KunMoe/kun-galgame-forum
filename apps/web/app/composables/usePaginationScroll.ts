import { useRoute } from '#imports'
import type { MaybeRefOrGetter, Ref } from 'vue'

export const usePaginationScroll = (
  page: Ref<number>,
  loading: MaybeRefOrGetter<boolean>,
  listScope: MaybeRefOrGetter<string> = ''
) => {
  const route = useRoute()
  const paginationStart = useTemplateRef<HTMLElement>('paginationStart')
  const requested = shallowRef<{ page: number; scope: string }>()
  const scope = computed(() =>
    JSON.stringify([
      route.path,
      Object.entries(route.query).filter(([key]) => key !== 'page'),
      toValue(listScope)
    ])
  )
  const isLoading = () => toValue(loading)

  const paginationPage = computed({
    get: () => page.value,
    set: (nextPage: number) => {
      if (nextPage === page.value) {
        return
      }

      requested.value = { page: nextPage, scope: scope.value }
      page.value = nextPage
    }
  })

  const isPaginationPending = computed(() => {
    const request = requested.value
    return (
      request?.page === page.value &&
      request.scope === scope.value &&
      isLoading()
    )
  })

  watch(
    [page, isLoading, scope],
    ([currentPage, pending, currentScope], [previousPage]) => {
      const request = requested.value
      if (!request) {
        return
      }
      if (
        request.scope !== currentScope ||
        (currentPage !== request.page && currentPage !== previousPage)
      ) {
        requested.value = undefined
        return
      }
      if (currentPage === request.page && !pending) {
        requested.value = undefined
        paginationStart.value?.scrollIntoView({
          behavior: 'smooth',
          block: 'start'
        })
      }
    },
    { flush: 'post' }
  )

  return { paginationPage, isPaginationPending }
}
