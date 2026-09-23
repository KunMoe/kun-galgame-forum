export const useNewsSources = () => {
  const asyncData = useApi('news-sources', (api, { signal }) =>
    api.GET('/news-sources', { signal })
  )
  const directory = computed<KunNewsSource[]>(
    () => asyncData.data.value?.items ?? []
  )
  const byKey = computed<Record<string, KunNewsSource>>(() =>
    Object.fromEntries(directory.value.map((source) => [source.key, source]))
  )
  const result = { directory, byKey, problem: asyncData.problem }
  return Object.assign(
    Promise.resolve(asyncData).then(() => result),
    result
  )
}
