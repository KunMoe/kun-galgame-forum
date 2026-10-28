import type { MetaFlatInput } from '@unhead/vue'

type KunSeoMetaInput = {
  title?: string
  description?: string
  keywords?: string
  ogType?: MetaFlatInput['ogType']
  ogImage?: string
  ogCard?: KunOgCardRef
  articleAuthor?: string[]
  articlePublishedTime?: string
  articleModifiedTime?: string
  // What useKunDisableSeo does, for a page that only knows once its data lands.
  noindex?: boolean
}

// Pages fetch lazily: a value read once in setup is still undefined during SSR,
// so the input is a getter the head re-reads as the data lands.
export const useKunSeoMeta = (source: MaybeRefOrGetter<KunSeoMetaInput>) => {
  const route = useRoute()
  const cardsEnabled = useRuntimeConfig().public.ogCardEnabled
  const site = kunOgSiteImage(cardsEnabled, kungal.domain.main)

  const meta = computed(() => {
    const input = toValue(source)
    const pageUrl = kunCanonicalUrl(route)
    if (input.noindex) {
      return {
        pageTitle: input.title,
        title: input.title ?? '',
        description: '',
        keywords: undefined,
        robots: 'noindex, nofollow',
        pageUrl,
        ogType: 'website',
        image: '',
        imageWidth: false,
        imageHeight: false,
        articleAuthor: undefined,
        articlePublishedTime: undefined,
        articleModifiedTime: undefined
      } as const
    }

    const title = input.title
      ? `${input.title} - ${kungal.title}`
      : kungal.title
    const cardUrl =
      input.ogCard && cardsEnabled
        ? `${kungal.domain.main}${kunOgCardPath(input.ogCard.kind, input.ogCard.id)}`
        : ''
    const image = cardUrl || input.ogImage || site.url
    // A caller's own image is of unknown size, so the site-wide width/height must be cleared
    // rather than left to describe a different picture.
    const size = cardUrl ? KUN_OG_CARD_SIZE : input.ogImage ? null : site
    return {
      pageTitle: input.title,
      title,
      description: input.description,
      keywords: input.keywords ?? kungal.keywords.toString(),
      robots: undefined,
      pageUrl,
      ogType: input.ogType ?? 'website',
      image,
      imageWidth: size ? size.width : false,
      imageHeight: size ? size.height : false,
      articleAuthor: input.articleAuthor,
      articlePublishedTime: input.articlePublishedTime,
      articleModifiedTime: input.articleModifiedTime
    } as const
  })

  const title = () => meta.value.title
  const description = () => meta.value.description
  const image = () => meta.value.image

  // The app's titleTemplate appends the site name; handing it the full title
  // put the name in <title> twice.
  useSeoMeta({
    title: () => meta.value.pageTitle,
    description,
    keywords: () => meta.value.keywords,
    robots: () => meta.value.robots,
    ogUrl: () => meta.value.pageUrl,
    ogType: () => meta.value.ogType,
    ogTitle: title,
    ogDescription: description,
    ogImage: image,
    ogImageAlt: title,
    ogImageWidth: () => meta.value.imageWidth,
    ogImageHeight: () => meta.value.imageHeight,
    twitterCard: 'summary_large_image',
    twitterTitle: title,
    twitterDescription: description,
    twitterImage: image,
    twitterImageAlt: title,
    articleAuthor: () => meta.value.articleAuthor,
    articlePublishedTime: () => meta.value.articlePublishedTime,
    articleModifiedTime: () => meta.value.articleModifiedTime
  })

  useHead({
    link: [{ rel: 'canonical', href: () => meta.value.pageUrl }]
  })
}
