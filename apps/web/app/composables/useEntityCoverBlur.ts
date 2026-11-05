import type { InjectionKey } from 'vue'

const KEY: InjectionKey<ComputedRef<boolean>> = Symbol('kun-entity-cover-blur')

export const provideEntityCoverBlur = (blur: MaybeRefOrGetter<boolean>) => {
  provide(
    KEY,
    computed(() => toValue(blur))
  )
}

export const useEntityCoverBlur = () => {
  const blur = inject(KEY, null)
  return computed(() => blur?.value ?? false)
}
