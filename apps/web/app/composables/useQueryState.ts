import { useRoute, useRouter } from '#imports'

// Nuxt's useRoute is scoped to the page and holds still while that page
// leaves. vue-router's global route moves the moment a link is clicked: the
// list being left saw ?page= and every filter fall back to their defaults,
// refetched page 1 and drew it until the next page's data arrived
// (「非第一页点卡片会先回到第一页再跳转」).
export const useQueryState = <S extends QuerySchema>(
  schema: S,
  options: { pageKey?: keyof S & string } = {}
) =>
  createQueryState(schema, {
    route: useRoute(),
    router: useRouter(),
    ...options
  })

export const useQueryWriter = () => {
  const route = useRoute()
  const router = useRouter()
  return (patch: Record<string, string | undefined>) =>
    writeQuery(router, route, patch)
}
