import { storeToRefs } from 'pinia'

export const useAifyLink = () =>
  computed(() => String(useRuntimeConfig().public.aifyLink || '').trim())

export const useAdVisible = () => {
  const { roles } = storeToRefs(usePersistUserStore())
  const link = useAifyLink()
  return computed(() => roles.value.length === 0 && link.value !== '')
}
