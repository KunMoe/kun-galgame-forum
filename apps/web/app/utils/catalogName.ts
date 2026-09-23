import { catalogEntityName, type CatalogName } from '#shared/utils/catalogName'

export const catalogNameText = (
  name: CatalogName,
  preferOriginal: boolean
): string => catalogEntityName(name, preferOriginal).name
