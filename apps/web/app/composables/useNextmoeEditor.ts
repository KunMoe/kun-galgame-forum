export const useNextmoeEditor = () => {
  const config = useRuntimeConfig()
  const route = useRoute()
  const userStore = usePersistUserStore()

  const editorUrl = (path: string, returnPath = route.fullPath) => {
    const params = new URLSearchParams({
      from: 'kungal',
      return: `${kungal.domain.main}${returnPath}`
    })
    if (userStore.sub) {
      params.set('login_hint', userStore.sub)
    }
    const base = String(config.public.editorUrl).replace(/\/+$/, '')
    return `${base}${path}?${params.toString()}`
  }

  const workEditUrl = (workId: number | string, returnPath?: string) =>
    editorUrl(`/works/${workId}/edit`, returnPath)

  const workHistoryUrl = (workId: number | string, returnPath?: string) =>
    editorUrl(`/history/work/${workId}`, returnPath)

  return { editorUrl, workEditUrl, workHistoryUrl }
}
