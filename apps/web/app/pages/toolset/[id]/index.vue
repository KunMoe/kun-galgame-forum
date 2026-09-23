<script setup lang="ts">
import type {
  WithContext,
  SoftwareApplication,
  Person,
  AggregateRating,
  DiscussionForumPosting
} from 'schema-dts'
import type { Toolset, WallComment } from '#shared/utils/api/schemas'
import { documentPlainText } from '#shared/utils/content/plainText'
import { contentPlainText } from '~/utils/contentPlainText'
import { deletedUserName, toKunUser } from '~/utils/userRef'
import { problemMessage } from '#shared/utils/api/message'

definePageMeta({ key: (route) => route.path })

const route = useRoute()
const id = computed(() => (route.params as { id: string }).id)

const { data, problem } = await useApi<Toolset>(
  () => `toolset:${id.value}`,
  (api, { signal }) =>
    api.GET('/toolsets/{toolset_id}', {
      params: { path: { toolset_id: id.value } },
      signal
    })
)

const { data: comments } = await useApi<WallComment[]>(
  () => `toolset-comments:${data.value?.id ?? 'none'}`,
  async (api) => {
    if (!data.value || data.value.comment_count === 0) {
      return { data: [], response: new Response(null, { status: 200 }) }
    }
    const page = await api.GET('/wall-comments', {
      params: {
        query: {
          subject_type: 'toolset',
          subject_id: data.value.id,
          limit: 5
        }
      }
    })
    return page.data
      ? { data: page.data.items, response: page.response }
      : { error: page.error, response: page.response }
  }
)

const toolset = data.value

if (toolset) {
  const author = toKunUser(toolset.author)
  const title = `${toolset.title} 资源下载`
  const pageUrl = `${kungal.domain.main}${route.path}`
  const description = truncateRunes(contentPlainText(toolset.content), 175)

  const osMap: Record<string, string> = {
    windows: 'Windows',
    linux: 'Linux',
    mac: 'macOS',
    macos: 'macOS'
  }
  const operatingSystem =
    osMap[(toolset.platform || '').toLowerCase()] || toolset.platform || 'All'

  const jsonLdApp: WithContext<
    SoftwareApplication & { aggregateRating?: AggregateRating }
  > = {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: title,
    alternateName: toolset.aliases || [],
    url: pageUrl,
    description,
    applicationCategory: toolset.toolset_type,
    operatingSystem,
    softwareVersion: toolset.release_channel,
    inLanguage: toolset.interface_language,
    datePublished: new Date(toolset.created_at).toISOString(),
    dateModified: new Date(toolset.updated_at).toISOString(),
    author: {
      '@type': 'Person',
      name: author.name
    } as Person,
    sameAs: (toolset.homepage_urls || []).slice(0, 5),
    interactionStatistic: [
      {
        '@type': 'InteractionCounter',
        interactionType: { '@type': 'WatchAction' },
        userInteractionCount: toolset.view_count || 0
      },
      {
        '@type': 'InteractionCounter',
        interactionType: { '@type': 'DownloadAction' },
        userInteractionCount: toolset.download_count || 0
      }
    ],
    ...(toolset.practicality_average && toolset.practicality_count
      ? {
          aggregateRating: {
            '@type': 'AggregateRating',
            ratingValue: Number(toolset.practicality_average.toFixed(2)),
            ratingCount: toolset.practicality_count,
            reviewCount: toolset.comment_count || toolset.practicality_count,
            bestRating: 5,
            worstRating: 1
          } as AggregateRating
        }
      : {})
  }

  useHead({
    script: [
      {
        id: 'schema-org-software-app',
        type: 'application/ld+json',
        innerHTML: jsonLdApp
      }
    ]
  })

  if (toolset.comment_count && toolset.comment_count > 0) {
    const forumJsonLd: WithContext<DiscussionForumPosting> = {
      '@context': 'https://schema.org',
      '@type': 'DiscussionForumPosting',
      url: pageUrl,
      headline: `${title} - Discussion`,
      articleBody: description,
      datePublished: new Date(toolset.created_at).toISOString(),
      dateModified: new Date(toolset.updated_at).toISOString(),
      author: { '@type': 'Person', name: author.name } as Person,
      commentCount: toolset.comment_count,
      comment: (comments.value ?? [])
        .filter((comment) => comment.state === 'visible')
        .map((comment) => ({
          '@type': 'Comment',
          text: truncateRunes(
            documentPlainText(comment.content, deletedUserName),
            280
          ),
          datePublished: new Date(comment.created_at).toISOString(),
          author: {
            '@type': 'Person',
            name: comment.author.name ?? deletedUserName
          } as Person
        }))
    }

    useHead({
      script: [
        {
          id: 'schema-org-discussion',
          type: 'application/ld+json',
          innerHTML: forumJsonLd
        }
      ]
    })
  }

  useKunSeoMeta({
    title,
    description,
    articleAuthor: [`${kungal.domain.main}/user/${toolset.author.id}`],
    articlePublishedTime: toolset.created_at,
    articleModifiedTime: toolset.updated_at
  })
} else {
  useKunDisableSeo('未找到该工具资源')
}
</script>

<template>
  <div>
    <ToolsetDetail v-if="data" :toolset="data" :id="id" />
    <KunNull
      v-else-if="problem"
      :description="problemMessage(problem)"
    />
    <KunNull v-else description="未找到该工具资源" />
  </div>
</template>
