import type { InjectionKey } from 'vue'
import type { TraitPage } from '#shared/utils/api/schemas'

const createTraitRoots = () => {
  const { allowsNsfw, stanceKey } = useContentStance()
  return useKunApi<TraitPage>(
    () => `trait-roots:${stanceKey.value}`,
    (api, { signal }) =>
      api.GET('/traits', {
        params: { query: { include_nsfw: allowsNsfw.value } },
        signal
      })
  )
}

const TRAIT_ROOTS: InjectionKey<ReturnType<typeof createTraitRoots>> =
  Symbol('trait-roots')

// A page awaits this, not the trait browser: the tree's key comes from the
// roots, and SSR does not refetch when a key changes after setup.
export const provideTraitRoots = () => {
  const roots = createTraitRoots()
  provide(TRAIT_ROOTS, roots)
  return roots
}

export const useTraitRoots = () => {
  const roots = inject(TRAIT_ROOTS)
  if (!roots) {
    throw new Error('useTraitRoots outside a page that provides them')
  }
  return roots
}
