<script setup lang="ts">
import type { VideoGame, WithContext, Person, BreadcrumbList } from 'schema-dts'

definePageMeta({ key: (route) => route.path })

const route = useRoute()

const { allowsNsfw } = useContentStance()

const gid = computed(() => {
  return parseInt((route.params as { gid: string }).gid)
})

const { data } = await useKunFetch<GalgameDetail>(`/galgame/${gid.value}`, {
  method: 'GET',
  watch: false,
  query: { galgame_id: gid.value }
})

if (data.value?.moved_to) {
  await navigateTo(`/galgame/${data.value.moved_to}`, {
    redirectCode: 301,
    replace: true
  })
}

const galgame = data.value?.moved_to ? null : data.value
const isShowGalgame = ref(true)

if (galgame) {
  const nsfw = galgame.content_limit === 'nsfw'
  // Being signed in used to be enough on its own, which showed every NSFW
  // detail page to a reader who had never asked for one. The account's stance
  // decides now; 模糊 lets the page through and masks the imagery instead.
  if (nsfw && !allowsNsfw.value) {
    isShowGalgame.value = false
  }

  if (!galgame.indexed || nsfw) {
    useKunDisableSeo(galgame.name)
  } else {
    const titleBase = galgame.name
    const original = galgame.name_original
    const title = original ? `${titleBase} | ${original}` : titleBase
    const pageUrl = `${kungal.domain.main}${route.path}`

    const developer = galgame.official[0]?.name
    const releaseYear = galgame.release_date
      ? new Date(galgame.release_date).getFullYear()
      : undefined
    const platformText = galgame.platform.slice(0, 3).join('、')
    // The API hands over every spoiler level so the tag panel can filter on the
    // client; anything a search engine indexes has to stay at level 0.
    const safeTags = galgame.tag.filter((t) => t.spoiler_level === 0)
    const contentGenres = safeTags
      .filter((t) => t.category === 'content')
      .map((t) => t.name)
    const fallbackDescription =
      `《${titleBase}》是一款${developer ? `由 ${developer} 开发的 ` : ''}Galgame（视觉小说）` +
      `${releaseYear ? `，${releaseYear} 年发售` : ''}` +
      `${platformText ? `，登陆 ${platformText}` : ''}` +
      `${contentGenres.length ? `，题材包括${contentGenres.slice(0, 3).join('、')}` : ''}` +
      `。本页收录其基本资料、制作 Staff、登场角色与声优, 以及玩家评分与评价。`

    const introText = truncateRunes(markdownToText(galgame.intro_text), 175)
    const description = introText || fallbackDescription

    const jsonLd: WithContext<VideoGame> = {
      '@context': 'https://schema.org',
      '@type': 'VideoGame',
      name: titleBase,
      alternateName: galgame.alias,
      url: pageUrl,
      image: getEffectiveBanner(galgame),
      description: description,
      inLanguage: galgame.original_language,
      datePublished:
        galgame.release_date || new Date(galgame.created).toISOString(),
      dateModified: new Date(galgame.updated).toISOString(),
      publisher: galgame.official.map((o) => ({
        '@type': 'Organization',
        name: o.name
      })),

      genre: contentGenres,
      keywords: safeTags
        .filter((t) => t.category === 'technical')
        .map((t) => t.name)
        .join(', '),

      ...(galgame.platform.length && { gamePlatform: galgame.platform }),

      ...(galgame.rating_count && {
        aggregateRating: {
          '@type': 'AggregateRating',
          ratingValue: Number((galgame.rating ?? 0).toFixed(1)),
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
          userInteractionCount: galgame.view
        }
      ],

      author: {
        '@type': 'Person',
        name: galgame.user.name
      } satisfies Person,
      contributor: galgame.contributor.map((c) => ({
        '@type': 'Person',
        name: c.name
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

    useHead({
      script: [
        {
          id: 'schema-org-video-game',
          type: 'application/ld+json',
          innerHTML: jsonLd
        },
        {
          id: 'schema-org-breadcrumb',
          type: 'application/ld+json',
          innerHTML: breadcrumbLd
        }
      ]
    })

    useKunSeoMeta({
      title,
      description,
      ogCard: { kind: 'galgame', id: galgame.id },
      articleAuthor: [`${kungal.domain.main}/user/${galgame.user.id}`],
      articlePublishedTime: galgame.created.toString(),
      articleModifiedTime: galgame.updated.toString()
    })
  }
} else {
  useKunDisableSeo('请求 Galgame 错误')
}
</script>

<template>
  <div>
    <div v-if="data && !data.moved_to">
      <Galgame v-if="isShowGalgame" :galgame="data" />

      <KunNsfwGate v-else noun="Galgame" @reveal="isShowGalgame = true" />
    </div>

    <KunNull v-else-if="!data?.moved_to" description="未找到这个 Galgame" />
  </div>
</template>
