export const useRenameCost = () => {
  const { data } = useKunApi('account-prices', (api, { signal }) =>
    api.GET('/account-prices', { signal })
  )
  return computed(() => data.value?.rename_cost ?? null)
}
