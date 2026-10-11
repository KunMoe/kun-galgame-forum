import type { Image } from '#shared/utils/api/schemas'
import { settle } from '#shared/utils/api/problem'
import { MAX_IMAGE_SIZE } from '~/config/upload'
import {
  isShrinkableImage,
  renameImageForType,
  shrinkImageToFit
} from './shrinkImage'

export type KunImagePurpose = 'content' | 'message'
export type KunWorkEditImagePreset = 'cover' | 'screenshot'

// What goes into a body or a cover column. The reference ping that keeps an
// image alive on the image host only finds this token or a bare hash column,
// so the absolute URL an upload returns must never be persisted.
export const imageToken = (hash: string) => `/image/${hash}`

const fitImage = async (file: Blob, filename: string) => {
  if (!isShrinkableImage(file, MAX_IMAGE_SIZE)) {
    return { file, filename }
  }
  const shrunk = await shrinkImageToFit(file, MAX_IMAGE_SIZE)
  if (!shrunk) {
    return { file, filename }
  }
  useMessage(
    `图片超过 ${MAX_IMAGE_SIZE / 1024 / 1024}MB，已自动压缩后上传`,
    'info'
  )
  const renamed = renameImageForType(filename, shrunk.type)
  return {
    file: new File([shrunk], renamed, { type: shrunk.type }),
    filename: renamed
  }
}

const imageForm = async (
  file: Blob,
  filename: string,
  field: string,
  value: string
) => {
  const fitted = await fitImage(file, filename)
  const form = new FormData()
  form.append('file', fitted.file, fitted.filename)
  form.append(field, value)
  return form
}

export const uploadImage = async (
  file: Blob,
  purpose: KunImagePurpose,
  filename = 'image'
): Promise<Image | null> => {
  const form = await imageForm(file, filename, 'purpose', purpose)
  const result = await settle(
    useApiClient().POST('/images', {
      body: form as unknown as {
        file: string
        purpose: KunImagePurpose
      },
      bodySerializer: (body) => body
    })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return null
  }
  return result.data
}

export const uploadWorkEditImage = async (
  file: Blob,
  preset: KunWorkEditImagePreset,
  filename = 'image'
): Promise<Image | null> => {
  const form = await imageForm(file, filename, 'preset', preset)
  const result = await settle(
    useApiClient().POST('/work-edit-images', {
      body: form as unknown as {
        file: string
        preset: KunWorkEditImagePreset
      },
      bodySerializer: (body) => body
    })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return null
  }
  return result.data
}

// A news banner has to go up through this face, never uploadImage: NextMoe
// keeps an image alive per uploading site, and refuses a banner hash that its
// news site does not hold.
export const uploadNewsSubmissionImage = async (
  file: Blob,
  filename = 'banner'
): Promise<Image | null> => {
  const form = new FormData()
  form.append('file', file, filename)
  const result = await settle(
    useApiClient().POST('/me/news-submission-images', {
      body: form as unknown as { file: string },
      bodySerializer: (body) => body
    })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return null
  }
  return result.data
}
