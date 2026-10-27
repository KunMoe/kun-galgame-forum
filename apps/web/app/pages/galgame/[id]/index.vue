<script setup lang="ts">
import type { VideoGame, WithContext, Person, BreadcrumbList } from 'schema-dts'
import type { Work } from '#shared/utils/api/schemas'
import { mergedInto } from '#shared/utils/api/merged'
import {
  deletionReasonLabel,
  entityDeletion,
  type EntityDeletion
} from '#shared/utils/api/deleted'
import { problemMessage } from '#shared/utils/api/message'
import { provideGalgamePage } from '~/composables/galgame/useGalgamePage'
import {
  catalogVocabularyName,
  pickCatalogIntro
} from '#shared/utils/catalogName'
import { deletedUserName } from '#shared/utils/deletedUser'
import { resourcePlatformLabel } from '~~/shared/utils/galgameResourceVocab'

definePageMeta({ key: (route) => route.path })

const route = useRoute()

const { allowsNsfw, stanceKey } = useContentStance()
const nameOf = useWorkName()
const namesOf = useCatalogName()

const workId = computed(() => String((route.params as { id: string }).id))
const { ratings, walkthroughs } = provideGalgamePage(workId.value)
const firstScreenPending = computed(
  () => ratings.pending.value || walkthroughs.pending.value
)

const movedTo = ref<number | null>(null)
const deletion = useState<EntityDeletion | null>(
  `work-deletion:${workId.value}`,
  () => null
)
const deletionText = computed(() => {
  const d = deletion.value
  if (!d) {
    return ''
  }
  const when = d.at ? ` (${d.at.slice(0, 10)})` : ''
  return `这部作品已被删除: ${deletionReasonLabel(d.reason)}${when}`
})
const { data, problem, pending } = await useKunApi<Work>(
  () => `work:${workId.value}:${stanceKey.value}`,
  async (api, { signal }) => {
    const res = await api.GET('/works/{work_id}', {
      params: {
        path: { work_id: workId.value },
        query: { include_nsfw: allowsNsfw.value }
      },
      signal
    })
    movedTo.value = mergedInto(res.error)
    deletion.value = entityDeletion(res.error)
    return res
  }
)

const redirectToMerged = (id: number) =>
  navigateTo(`/galgame/${id}`, { redirectCode: 301, replace: true })

if (movedTo.value) {
  await redirectToMerged(movedTo.value)
} else if (deletion.value && import.meta.server) {
  setResponseStatus(useRequestEvent()!, 410)
}
watch(movedTo, (id) => {
  if (id) {
    redirectToMerged(id)
  }
})
useKunNotFound(
  () =>
    problem.value?.status === 404 && !movedTo.value && !deletion.value,
  '未找到这个 Galgame'
)

// Being signed in used to be enough on its own, which showed every NSFW
// detail page to a reader who had never asked for one. The account's stance
// decides now; 模糊 lets the page through and masks the imagery instead.
// It was also decided once at setup, and setup can run before the data lands
// (a hydration key miss): the gate never closed and an NSFW work rendered for a
// hide-stance account (2026-09-24). So it follows data and stance.
const revealed = ref(false)
const isShowGalgame = computed(
  () => !data.value?.is_nsfw || allowsNsfw.value || revealed.value
)

const indexable = computed(() => {
  const galgame = data.value
  return galgame && galgame.is_published && !galgame.is_nsfw ? galgame : null
})

const seo = computed(() => {
  const galgame = indexable.value
  if (!galgame) {
    return null
  }
  const { name: titleBase, original } = namesOf(galgame)
  const title = original ? `${titleBase} | ${original}` : titleBase
  const pageUrl = `${kungal.domain.main}${route.path}`

  const developer = galgame.companies[0]
    ? namesOf(galgame.companies[0]).name
    : ''
  const releaseYear = galgame.release_date
    ? new Date(galgame.release_date).getFullYear()
    : undefined
  const platformText = galgame.resource_platforms
    .slice(0, 3)
    .map((platform) => resourcePlatformLabel(platform))
    .join('、')
  // The API hands over every spoiler level so the tag panel can filter on the
  // client; anything a search engine indexes has to stay at none.
  const safeTags = galgame.tags.filter((t) => t.spoiler === 'none')
  const contentGenres = safeTags
    .filter((t) => t.tag_kind === 'content')
    .map((t) => catalogVocabularyName(t))
  const fallbackDescription =
    `《${titleBase}》是一款${developer ? `由 ${developer} 开发的 ` : ''}Galgame（视觉小说）` +
    `${releaseYear ? `，${releaseYear} 年发售` : ''}` +
    `${platformText ? `，登陆 ${platformText}` : ''}` +
    `${contentGenres.length ? `，题材包括${contentGenres.slice(0, 3).join('、')}` : ''}` +
    `。本页收录其基本资料、制作 Staff、登场角色与声优, 以及玩家评分与评价。`

  const introText = truncateRunes(
    markdownToText(pickCatalogIntro(galgame.intros)?.value ?? ''),
    175
  )
  const description = introText || fallbackDescription

  const jsonLd: WithContext<VideoGame> = {
    '@context': 'https://schema.org',
    '@type': 'VideoGame',
    name: titleBase,
    alternateName: [...(original ? [original] : []), ...galgame.aliases],
    url: pageUrl,
    image: galgame.banner?.url || galgame.cover?.url,
    description: description,
    inLanguage: galgame.original_language ?? undefined,
    datePublished: galgame.release_date || galgame.created_at,
    dateModified: galgame.updated_at,
    publisher: galgame.companies.map((company) => ({
      '@type': 'Organization',
      name: namesOf(company).name
    })),

    genre: contentGenres,
    keywords: safeTags
      .filter((t) => t.tag_kind === 'meta')
      .map((t) => catalogVocabularyName(t))
      .join(', '),

    ...(galgame.resource_platforms.length && {
      gamePlatform: galgame.resource_platforms.map((platform) =>
        resourcePlatformLabel(platform)
      )
    }),

    ...(galgame.rating_count &&
      galgame.rating_score != null && {
        aggregateRating: {
          '@type': 'AggregateRating',
          ratingValue: Number(galgame.rating_score.toFixed(1)),
          ratingCount: galgame.rating_count,
          bestRating: 10,
          worstRating: 1
        }
      }),

    interactionStatistic: [
      {
        '@type': 'InteractionCounter',
        interactionType: {
          '@type': 'LikeAction'
        },
        userInteractionCount: galgame.like_count
      },
      {
        '@type': 'InteractionCounter',
        interactionType: {
          '@type': 'WatchAction'
        },
        userInteractionCount: galgame.view_count
      }
    ],

    ...(galgame.creator
      ? {
          author: {
            '@type': 'Person',
            name: galgame.creator.name ?? deletedUserName
          } satisfies Person
        }
      : {}),
    contributor: galgame.contributors.map((c) => ({
      '@type': 'Person',
      name: c.name ?? deletedUserName
    })) satisfies Person[]
  }

  const breadcrumbLd: WithContext<BreadcrumbList> = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      {
        '@type': 'ListItem',
        position: 1,
        name: '首页',
        item: kungal.domain.main
      },
      {
        '@type': 'ListItem',
        position: 2,
        name: 'Galgame',
        item: `${kungal.domain.main}/galgame`
      },
      { '@type': 'ListItem', position: 3, name: titleBase, item: pageUrl }
    ]
  }

  return { galgame, title, description, jsonLd, breadcrumbLd }
})

useHead(() => ({
  script: seo.value
    ? [
        {
          id: 'schema-org-video-game',
          type: 'application/ld+json',
          innerHTML: seo.value.jsonLd
        },
        {
          id: 'schema-org-breadcrumb',
          type: 'application/ld+json',
          innerHTML: seo.value.breadcrumbLd
        }
      ]
    : []
}))

useKunSeoMeta(() => {
  if (seo.value) {
    const { galgame, title, description } = seo.value
    return {
      title,
      description,
      ogCard: { kind: 'galgame', id: Number(galgame.id) },
      ...(galgame.creator
        ? {
            articleAuthor: [`${kungal.domain.main}/user/${galgame.creator.id}`]
          }
        : {}),
      articlePublishedTime: galgame.created_at,
      articleModifiedTime: galgame.updated_at
    }
  }
  if (data.value) {
    return { noindex: true, title: nameOf(data.value) }
  }
  return problem.value ? { noindex: true, title: '请求 Galgame 错误' } : {}
})
</script>

<template>
  <div>
    <div v-if="data && !firstScreenPending">
      <Galgame v-if="isShowGalgame" :galgame="data" />

      <KunNsfwGate v-else noun="Galgame" @reveal="revealed = true" />
    </div>

    <GalgameDetailSkeleton v-else-if="pending || (data && firstScreenPending)" />

    <KunNull v-else-if="deletion" :description="deletionText" />

    <KunNull v-else-if="problem" :description="problemMessage(problem)" />

    <KunNull v-else description="未找到这个 Galgame" />
  </div>
</template>
