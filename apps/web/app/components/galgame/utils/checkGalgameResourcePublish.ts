import {
  kunGalgameResourceTypeOptions,
  kunGalgameResourceLanguageOptions,
  kunGalgameResourcePlatformOptions
} from '~/constants/galgame'
import type { GalgameResourceStoreTemp } from '~/store/types/galgame/resource'

export const checkGalgameResourcePublish = (link: GalgameResourceStoreTemp) => {
  if (
    !kunGalgameResourceTypeOptions
      .map((type) => type.value as string)
      .filter((item) => item !== 'all')
      .includes(link.type)
  ) {
    useMessage(10556, 'warn')
    return false
  }

  if (!link.link.length || link.link.length > 20) {
    useMessage(10557, 'warn')
    return false
  }

  for (const l of link.link) {
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
    !kunGalgameResourceLanguageOptions
      .map((lang) => lang.value as string)
      .filter((item) => item !== 'all')
      .includes(link.language)
  ) {
    useMessage(10560, 'warn')
    return false
  }

  if (
    !kunGalgameResourcePlatformOptions
      .map((platform) => platform.value as string)
      .filter((item) => item !== 'all')
      .includes(link.platform)
  ) {
    useMessage(10561, 'warn')
    return false
  }

  if (!ResourceSizePattern.test(link.size)) {
    useMessage(10562, 'warn')
    return false
  }

  if (link.code.length > 1007) {
    useMessage(10563, 'warn')
    return false
  }

  if (link.password.length > 1007) {
    useMessage(10564, 'warn')
    return false
  }

  if (link.note.length > 10000) {
    useMessage(10565, 'warn')
    return false
  }

  return true
}
