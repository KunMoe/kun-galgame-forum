import type {
  KunEditorAdapters,
  MentionUser,
  StickerPack
} from '@kungal/editor-core'

export const useKunEditorAdapters = (opts?: {
  image?: boolean
}): KunEditorAdapters => {
  const allowImage = opts?.image !== false

  const uploadImage = async (file: File) => {
    const form = new FormData()
    form.append('image', file)
    const url = await kunFetch<string>('/image/topic', {
      method: 'POST',
      body: form,
      watch: false
    })
    if (!url) {
      throw new Error('图片上传失败')
    }
    return url
  }

  const searchMentionUsers = async (query: string): Promise<MentionUser[]> =>
    (await kunFetch<MentionUser[]>('/user/search', {
      query: { q: query, limit: 8 }
    })) ?? []

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
