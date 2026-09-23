import type { Image } from '#shared/utils/api/schemas'
import { settle } from '#shared/utils/api/problem'

export type KunImagePurpose = 'content' | 'message'
export type KunWorkEditImagePreset = 'cover' | 'screenshot'

// What goes into a body or a cover column. The reference ping that keeps an
// image alive on the image host only finds this token or a bare hash column,
// so the absolute URL an upload returns must never be persisted.
export const imageToken = (hash: string) => `/image/${hash}`

const imageForm = (file: Blob, filename: string, field: string, value: string) => {
  const form = new FormData()
  form.append('file', file, filename)
  form.append(field, value)
  return form
}

export const uploadImage = async (
  file: Blob,
  purpose: KunImagePurpose,
  filename = 'image'
): Promise<Image | null> => {
  const result = await settle(
    useApiClient().POST('/images', {
      body: imageForm(file, filename, 'purpose', purpose) as unknown as {
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
  const result = await settle(
    useApiClient().POST('/work-edit-images', {
      body: imageForm(file, filename, 'preset', preset) as unknown as {
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
