<script setup lang="ts">
import type {
  Article,
  WebSite,
  Organization,
  Review,
  Person,
  WithContext
} from 'schema-dts'
import { documentPlainText } from '#shared/utils/content/plainText'
import type { Website, WallComment } from '#shared/utils/api/schemas'
import { deletedUserName } from '~/utils/userRef'

definePageMeta({ key: (route) => route.path })

const route = useRoute()

const host = computed(() => (route.params as { domain: string }).domain)

const { data, refresh } = await useApi<Website>(
  () => `website:${host.value}`,
  (api) =>
    api.GET('/websites/{website_host}', {
      params: { path: { website_host: host.value } }
    })
)

const iconUrl = computed(
  () => data.value?.icon?.url ?? data.value?.external_icon_url ?? ''
)

const { data: reviews } = await useApi<WallComment[]>(
  () => `website-reviews:${data.value?.id ?? 'none'}`,
  async (api) => {
    if (!data.value || data.value.is_nsfw) {
      return { data: [], response: new Response(null, { status: 200 }) }
    }
    const page = await api.GET('/wall-comments', {
      params: {
        query: {
          subject_type: 'website',
          subject_id: data.value.id,
          limit: 20
        }
      }
    })
    return page.data
      ? { data: page.data.items, response: page.response }
      : { error: page.error, response: page.response }
  }
)

const jsonLd = computed<WithContext<Article> | null>(() => {
  if (!data.value) {
    return null
  }

  const website = data.value
  const pageUrl = `${kungal.domain.main}/website/${website.host}`

  const publisherSchema: Organization = {
    '@type': 'Organization',
    name: kungal.title,
    logo: {
      '@type': 'ImageObject',
      url: `${kungal.domain.main}/kungalgame.webp`
    }
  }

  const aboutWebsiteSchema: WebSite = {
    '@type': 'WebSite',
    name: website.title,
    url: website.host,
    description: website.description,
    inLanguage: website.language,
    isFamilyFriendly: !website.is_nsfw,
    image: iconUrl.value
  }

  const reviewsSchema: Review[] = (reviews.value ?? [])
    .filter((comment) => comment.state === 'visible')
    .map((comment) => ({
      '@type': 'Review',
      author: {
        '@type': 'Person',
        name: comment.author.name ?? deletedUserName,
        url: `${kungal.domain.main}/user/${comment.author.id}`
      } as Person,
      datePublished: comment.created_at,
      reviewBody: documentPlainText(comment.content, deletedUserName),
      publisher: publisherSchema
    }))

  const articleSchema: Article = {
    '@type': 'Article',
    mainEntityOfPage: pageUrl,
    headline: `关于 ${website.title} 的介绍与评价`,
    description: website.description,
    image: iconUrl.value,
    datePublished: website.created_at,
    dateModified: website.updated_at,
    author: publisherSchema,
    publisher: publisherSchema,
    about: aboutWebsiteSchema,
    keywords: [
      website.website_category.label,
      ...website.website_tags.map((t) => t.label)
    ].join(', '),
    ...(reviewsSchema.length > 0 && { review: reviewsSchema })
  }

  return {
    '@context': 'https://schema.org',
    ...articleSchema
  }
})

if (data.value) {
  if (!data.value.is_nsfw) {
    useHead({
      script: [
        {
          id: 'schema-org-website',
          type: 'application/ld+json',
          innerHTML: jsonLd.value
        }
      ]
    })

    useKunSeoMeta({
      title: data.value.title,
      description: data.value.description,
      ogImage: iconUrl.value,
      articlePublishedTime: data.value.created_at,
      articleModifiedTime: data.value.updated_at
    })
  } else {
    useKunDisableSeo(data.value.title)
  }
} else {
  useKunDisableSeo('未找到该网站')
}
</script>

<template>
  <div v-if="data" class="grid grid-cols-1 gap-3 lg:grid-cols-3">
    <div class="min-w-0 space-y-3 lg:col-span-2">
      <KunCard
        :is-transparent="false"
        :is-hoverable="false"
        class-name="p-6"
        content-class="space-y-6"
      >
        <div class="flex items-start space-x-6">
          <div class="flex-shrink-0">
            <KunImage
              :src="iconUrl"
              :alt="data.title"
              class="h-20 w-20 rounded-2xl object-cover"
            />
          </div>
          <div class="space-y-3">
            <h1 class="text-default-900 text-3xl font-bold">
              {{ data.title }}
            </h1>

            <div class="text-default-500 flex items-center space-x-6 text-sm">
              <div class="flex items-center space-x-1">
                <KunIcon name="lucide:eye" />
                <span>{{ formatNumber(data.view_count) }} 次访问</span>
              </div>
              <div class="flex items-center space-x-1">
                <KunIcon name="lucide:clock" />
                <span
                  >更新于 <KunTime :time="data.updated_at" type="date"
                /></span>
              </div>
            </div>
          </div>
        </div>

        <p class="text-default-600 text-lg leading-relaxed">
          {{ data.description }}
        </p>

        <WebsiteDetailTagVisualization :tags="data.website_tags" />

        <WebsiteOperation :website="data" @refresh="refresh" />
      </KunCard>

      <WebsiteCommentCommunityContainer :website-id="Number(data.id)" />
    </div>

    <div class="min-w-0 space-y-3">
      <WebsiteDetailInfo :data="data" />

      <KunCard :is-transparent="false" :is-hoverable="false" class-name="p-6">
        <h3 class="text-default-900 mb-4 text-lg font-semibold">相关标签</h3>
        <div class="flex flex-wrap gap-2">
          <WebsiteTag :tags="data.website_tags" :is-nav="true" />
        </div>
      </KunCard>

      <WebsiteDetailStats :data="data" />
    </div>
  </div>
</template>
