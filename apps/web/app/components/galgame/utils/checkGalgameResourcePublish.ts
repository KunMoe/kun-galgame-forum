import type { GalgameResourceStoreTemp } from '~/store/types/galgame/resource'
import { parseResourceSize } from '~~/shared/utils/resourceSize'
import {
  RESOURCE_TYPE_LABELS,
  LANGUAGE_LABELS,
  PLATFORM_LABELS,
  RUNTIME_LABELS,
  hasRuntimeAxis
} from '~~/shared/utils/galgameResourceVocab'

export const checkGalgameResourcePublish = (link: GalgameResourceStoreTemp) => {
  if (!RESOURCE_TYPE_LABELS[link.resource_type]) {
    useMessage(10556, 'warn')
    return false
  }

  if (!link.download_urls.length || link.download_urls.length > 20) {
    useMessage(10557, 'warn')
    return false
  }

  for (const l of link.download_urls) {
    if (l.trim().length > 1007) {
      useMessage(10558, 'warn')
      return false
    }

    if (!isValidURL(l.trim())) {
      useMessage(10559, 'warn')
      return false
    }
  }

  if (
    !link.resource_languages.length ||
    link.resource_languages.some((k) => !LANGUAGE_LABELS[k])
  ) {
    useMessage(10560, 'warn')
    return false
  }

  if (
    (!link.resource_platforms.length && !link.resource_runtimes.length) ||
    link.resource_platforms.some((k) => !PLATFORM_LABELS[k])
  ) {
    useMessage(10561, 'warn')
    return false
  }

  if (hasRuntimeAxis(link.resource_type)) {
    if (
      !link.resource_runtimes.length ||
      link.resource_runtimes.some((k) => !RUNTIME_LABELS[k])
    ) {
      useMessage(10570, 'warn')
      return false
    }
  }

  if (!parseResourceSize(link.size)) {
    useMessage(10562, 'warn')
    return false
  }

  if (link.extraction_code.length > 1007) {
    useMessage(10563, 'warn')
    return false
  }

  if (link.archive_password.length > 1007) {
    useMessage(10564, 'warn')
    return false
  }

  if (link.content_markdown.length > 10000) {
    useMessage(10565, 'warn')
    return false
  }

  return true
}
