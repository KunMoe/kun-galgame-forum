import {
  DEFAULT_IMAGE_CDN_BASE,
  imageHashUrl
} from '#shared/utils/imageService'

export const imageCdnBase = (): string => {
  try {
    const v = useRuntimeConfig().public.imageCdnBase as string | undefined
    if (v) return v.replace(/\/+$/, '')
  } catch {
    // useRuntimeConfig throws outside a Nuxt context (plain unit tests, module
    // scope). Fall through to the shared default.
  }
  return DEFAULT_IMAGE_CDN_BASE
}

export const imageTokenUrl = (tokenOrHash: string): string => {
  if (!tokenOrHash || tokenOrHash.startsWith('http')) return tokenOrHash
  const hash = tokenOrHash.startsWith('/image/')
    ? tokenOrHash.slice('/image/'.length)
    : tokenOrHash
  return imageHashUrl(imageCdnBase(), hash)
}
