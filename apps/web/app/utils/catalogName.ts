import type { WorkRef } from '#shared/utils/api/schemas'

type CatalogName = Pick<WorkRef, 'display_name' | 'latin' | 'localized'>

const ZH_LOCALES = ['zh-Hans', 'zh', 'zh-Hant']

export const catalogNameText = (
  name: CatalogName,
  preferOriginal: boolean
): string => {
  const zh = ZH_LOCALES.map((tag) => name.localized[tag]?.value).find(Boolean)
  return preferOriginal
    ? name.display_name || zh || name.latin || ''
    : zh || name.display_name || name.latin || ''
}
