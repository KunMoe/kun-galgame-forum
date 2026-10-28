<script setup lang="ts">
import { isMissingProblem } from '#shared/utils/api/problem'
import type {
  Review,
  WithContext,
  Person,
  Organization,
  VideoGame,
  Rating as SchemaRating,
  PropertyValue
} from 'schema-dts'
import { aspectDims } from '~/utils/galgame/ratingCard'

definePageMeta({ key: (route) => route.path })

const route = useRoute()
const id = computed(() => (route.params as { id: string }).id)
const nameOf = useCatalogName()

const { data, problem, pending, refresh } = await useKunApi(
  () => `rating:${id.value}`,
  (api, { signal }) =>
    api.GET('/ratings/{rating_id}', {
      params: { path: { rating_id: id.value } },
      signal
    }),
  { critical: true }
)
useKunNotFound(() => isMissingProblem(problem.value), '未找到这条评价')

const jsonLd = computed<WithContext<Review> | null>(() => {
  if (!data.value) {
    return null
  }

  const rating = data.value
  const work = rating.work_summary
  const author = toKunUser(rating.author)
  const titleBase = nameOf(work).name
  const pageUrl = `${kungal.domain.main}${route.path}`
  const gameUrl = `${kungal.domain.main}/galgame/${work.id}`
  const dims = aspectDims(rating.aspect_scores)

  const publisherSchema: Organization = {
    '@type': 'Organization',
    name: kungal.title,
    logo: {
      '@type': 'ImageObject',
      url: `${kungal.domain.main}/kungalgame.webp`
    }
  }

  const authorSchema: Person = {
    '@type': 'Person',
    name: author.name,
    url: `${kungal.domain.main}/user/${author.id}`
  }

  const itemReviewedSchema: VideoGame = {
    '@type': 'VideoGame',
    name: titleBase,
    url: gameUrl,
    image: work.banner?.url,
    isFamilyFriendly: !work.is_nsfw,
    ...(work.maker && {
      publisher: { '@type': 'Organization', name: nameOf(work.maker).name }
    })
  }

  const reviewRatingSchema: SchemaRating = {
    '@type': 'Rating',
    ratingValue: rating.overall,
    bestRating: 10,
    worstRating: 1
  }

  const additionalProps: PropertyValue[] = [
    { name: '艺术风格', value: dims.art },
    { name: '故事情节', value: dims.story },
    { name: '音乐体验', value: dims.music },
    { name: '角色塑造', value: dims.character },
    { name: '路线设计', value: dims.route },
    { name: '系统交互', value: dims.system },
    { name: '声优演绎', value: dims.voice },
    { name: '重玩价值', value: dims.replay_value }
  ].map((p) => ({ '@type': 'PropertyValue', ...p }))

  return {
    '@context': 'https://schema.org',
    '@type': 'Review',
    mainEntityOfPage: pageUrl,
    headline: `${author.name} 对 ${titleBase} 的评价`,
    datePublished: rating.created_at,
    dateModified: rating.updated_at,
    author: authorSchema,
    publisher: publisherSchema,
    itemReviewed: itemReviewedSchema,
    reviewRating: reviewRatingSchema,
    reviewBody: truncateRunes(markdownToText(rating.short_summary || ''), 250),
    interactionStatistic: [
      {
        '@type': 'InteractionCounter',
        interactionType: { '@type': 'LikeAction' },
        userInteractionCount: rating.like_count
      },
      {
        '@type': 'InteractionCounter',
        interactionType: { '@type': 'WatchAction' },
        userInteractionCount: rating.view_count
      }
    ],
    additionalProperty: additionalProps
  }
})

useHead(() => ({
  script:
    jsonLd.value && !data.value?.work_summary.is_nsfw
      ? [
          {
            id: 'schema-org-galgame-rating',
            type: 'application/ld+json',
            innerHTML: jsonLd.value
          }
        ]
      : []
}))

useKunSeoMeta(() => {
  const rating = data.value
  if (!rating) {
    return problem.value
      ? { noindex: true, title: '请求 Galgame 评分数据错误' }
      : {}
  }
  const work = rating.work_summary
  const author = toKunUser(rating.author)
  if (work.is_nsfw) {
    return { noindex: true, title: `${author.name} 的评价` }
  }

  const { name: titleBase, original } = nameOf(work)
  const title =
    original && original !== titleBase
      ? `${author.name} 对 ${titleBase} (${original}) 的评价`
      : `${author.name} 对 ${titleBase} 的评价`

  const description = truncateRunes(
    rating.short_summary
      ? markdownToText(rating.short_summary)
      : `${author.name} 对 ${titleBase} 的评分 ${rating.overall}/10`,
    175
  )

  return {
    title,
    description,
    ogImage: work.banner?.url ?? '',
    articleAuthor: [`${kungal.domain.main}/user/${author.id}`],
    articlePublishedTime: rating.created_at,
    articleModifiedTime: rating.updated_at
  }
})
</script>

<template>
  <div>
    <GalgameRatingDetail v-if="data" :data="data" :refresh="refresh" />
    <GalgameRatingDetailSkeleton v-else-if="pending" />
    <KunNull v-else description="未找到该评分" />
  </div>
</template>
