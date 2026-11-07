import { imageHashUrl } from '#shared/utils/imageService'
import { imageCdnBase } from './imageSrc'

export const galgameImageSrc = (row: {
  cdn_url?: string
  image_hash: string
}): string => row.cdn_url || imageHashUrl(imageCdnBase(), row.image_hash)
