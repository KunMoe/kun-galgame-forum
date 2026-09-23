import type {
  KunEditorAdapters,
  MentionUser,
  StickerPack
} from '@kungal/editor-core'
import { settle } from '#shared/utils/api/problem'
import { toKunUser } from '~/utils/userRef'
import { imageToken, uploadImage as uploadSiteImage } from '~/utils/uploadImage'

export const useKunEditorAdapters = (opts?: {
  image?: boolean
}): KunEditorAdapters => {
  const allowImage = opts?.image !== false
  const api = useApiClient()

  const uploadImage = async (file: File) => {
    const image = await uploadSiteImage(file, 'content', file.name)
    if (!image) {
      throw new Error('图片上传失败')
    }
    return imageToken(image.hash)
  }

  const searchMentionUsers = async (query: string): Promise<MentionUser[]> => {
    const q = query.trim()
    if (!q) return []
    const result = await settle(
      api.GET('/users', {
        params: { query: { q, limit: 8 } }
      })
    )
    if (!result.ok) {
      reportProblem(result.problem)
      return []
    }
    return result.data.items.map(toKunUser)
  }

  // The packs come from sticker.kungal.com through our own cached server
  // route. The URLs it hands back are content-addressed, and they are what
  // gets written into the post -- the previous scheme wrote a path that named
  // a position in a mutable collection, which is why years of posts now
  // render broken images.
  const stickerSource = (): Promise<StickerPack[]> => useStickerPacks().load()

  const notify: KunEditorAdapters['notify'] = (message, level) => {
    useMessage(message, level)
  }

  return allowImage
    ? { uploadImage, searchMentionUsers, stickerSource, notify }
    : { searchMentionUsers, notify }
}
