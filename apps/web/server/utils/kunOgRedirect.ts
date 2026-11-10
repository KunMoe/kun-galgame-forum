import type { H3Event } from 'h3'
import { type KunOgCard, kunOgSignedUrl, kunOgSiteCard } from './kunOgCard'
import { kunRedirect } from './kunRedirect'

/**
 * Short, not immutable: the path stays the same for the life of an entity, so a renamed topic
 * has to be able to point at a different signed URL. The render behind that URL is the part
 * the renderer caches for a year.
 */
const CACHE_CONTROL = 'public, max-age=600, stale-while-revalidate=86400'

export const kunOgRedirect = (event: H3Event, card: KunOgCard | null) => {
  const origin = useRuntimeConfig().public.KUN_GALGAME_URL || ''
  const target =
    (card && kunOgSignedUrl(card)) ??
    kunOgSignedUrl(kunOgSiteCard(origin)) ??
    `${origin}/kungalgame.webp`

  setHeader(event, 'Cache-Control', CACHE_CONTROL)
  return kunRedirect(event, target, 302)
}
