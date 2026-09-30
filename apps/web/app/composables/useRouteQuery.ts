import { useRouteQuery as useVueUseRouteQuery } from '@vueuse/router'
import { useRoute, useRouter } from '#imports'

// vueuse reads vue-router's global route, which moves to the next page the
// moment a link is clicked. The list being left then saw ?page= and every
// filter fall back to their defaults, refetched page 1 and drew it until the
// next page's data arrived: 「非第一页点卡片会先回到第一页再跳转」. Nuxt's
// useRoute is scoped to the page and holds still while that page leaves.
export const useRouteQuery = ((
  name: string,
  defaultValue?: Parameters<typeof useVueUseRouteQuery>[1],
  options?: Parameters<typeof useVueUseRouteQuery>[2]
) =>
  useVueUseRouteQuery(name, defaultValue, {
    route: useRoute(),
    router: useRouter(),
    ...options
  })) as typeof useVueUseRouteQuery
