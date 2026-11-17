import { imageTokenUrl } from '~/utils/imageSrc'

export const useAgentCoverGate = () => {
  const { stance } = useContentStance()

  const coverOf = (c: AgentEntityCard): AgentCardCover | null => {
    const cover = c.cover
    if (typeof cover === 'string') {
      if (!cover) return null
      return gate(c.content_limit === 'nsfw', { src: cover, thumbhash: null })
    }
    if (!cover?.hash) return null
    const graded =
      c.content_limit === 'nsfw' ||
      cover.sexual === 'suggestive' ||
      cover.sexual === 'explicit'
    return gate(graded, {
      src: imageTokenUrl(cover.hash),
      thumbhash: cover.thumbhash
    })
  }

  const gate = (
    graded: boolean,
    cover: Omit<AgentCardCover, 'blurred'>
  ): AgentCardCover | null => {
    if (!graded) return { ...cover, blurred: false }
    if (stance.value === 'hide') return null
    return { ...cover, blurred: stance.value === 'blur' }
  }

  return { coverOf }
}
