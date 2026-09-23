import { storeToRefs } from 'pinia'
import type { CatalogName } from '#shared/utils/catalogName'

export const useCatalogName = () => {
  const { showKUNGalgamePreferOriginalName } = storeToRefs(
    usePersistSettingsStore()
  )
  return (n: CatalogName) =>
    catalogEntityName(n, showKUNGalgamePreferOriginalName.value)
}
