import { storeToRefs } from 'pinia'
import type { WorkRef } from '#shared/utils/api/schemas'

type NamedWork = Pick<WorkRef, 'display_name' | 'latin' | 'localized'>

const ZH_LOCALES = ['zh-Hans', 'zh', 'zh-Hant']

export const pickWorkName = (work: NamedWork, preferOriginal: boolean) => {
  const zh = ZH_LOCALES.map((tag) => work.localized[tag]?.value).find(
    (value) => !!value
  )
  if (preferOriginal) {
    return work.display_name || zh || work.latin || ''
  }
  return zh || work.display_name || work.latin || ''
}

export const useWorkName = () => {
  const { showKUNGalgamePreferOriginalName } = storeToRefs(
    usePersistSettingsStore()
  )
  return (work: NamedWork) =>
    pickWorkName(work, showKUNGalgamePreferOriginalName.value)
}
