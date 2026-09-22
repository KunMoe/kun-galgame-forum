export default defineNuxtPlugin((nuxtApp) => {
  const userStore = usePersistUserStore()

  watch(
    () => userStore.id,
    (id) => {
      if (!id) return
      nuxtApp.runWithContext(() => useCloudPreferences().sync())
    },
    { immediate: true }
  )
})
