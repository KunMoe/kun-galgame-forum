export const collectionDisplayName = (
  c: { is_default: boolean; title: string },
  ownerName?: string | null
): string => {
  if (c.is_default && !c.title) {
    return `${ownerName || '我'}的收藏夹`
  }
  return c.title
}

export const collectionDisplayDescription = (
  c: { is_default: boolean; description: string },
  ownerName?: string | null
): string => {
  if (c.is_default && !c.description) {
    const who = ownerName || '我'
    return `${who}所有收藏的游戏已经被放置在 ${who}的收藏夹`
  }
  return c.description
}
