import { storeToRefs } from 'pinia'
import type { WorkRef } from '#shared/utils/api/schemas'
import { catalogNameText } from '~/utils/catalogName'

export const useWorkName = () => {
  const { showKUNGalgamePreferOriginalName } = storeToRefs(
    usePersistSettingsStore()
  )
  return (work: Pick<WorkRef, 'display_name' | 'latin' | 'localized'>) =>
    catalogNameText(work, showKUNGalgamePreferOriginalName.value)
}
