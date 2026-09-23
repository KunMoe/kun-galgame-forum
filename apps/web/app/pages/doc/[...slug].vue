<script setup lang="ts">
import { contentHeadings } from '~/utils/contentPlainText'

const route = useRoute()

const docSlug = computed(() => {
  const slug = route.params.slug
  return Array.isArray(slug) ? slug.join('/') : slug || ''
})

const { data } = await useApi(
  () => `doc:${docSlug.value}`,
  (api, { signal }) =>
    api.GET('/docs/{doc_slug}', {
      params: { path: { doc_slug: docSlug.value } },
      signal
    })
)

const headings = computed(() => contentHeadings(data.value?.content))

if (data.value) {
  useKunSeoMeta({
    title: data.value.title,
    description: data.value.description,
    ogImage: data.value.banner?.url,
    ogType: 'article',
    articleAuthor: [`${kungal.domain.main}/user/${data.value.author.id}`],
    articlePublishedTime: data.value.published_at,
    articleModifiedTime: data.value.edited_at ?? undefined
  })
} else {
  useKunDisableSeo('未找到该文档')
}
</script>

<template>
  <div v-if="data" class="min-h-[calc(100dvh-6rem)] pb-6">
    <div class="flex">
      <DocDetailCategoryTree />

      <article class="min-w-0 flex-1 space-y-6 pl-0 lg:pr-67 xl:pl-67">
        <DocDetailHeader :metadata="data" />
        <ContentDocument :document="data.content" />
        <DocDetailFooter />
      </article>

      <div v-if="headings.length" class="hidden lg:block">
        <div class="fixed -translate-x-67">
          <DocDetailTableOfContent :links="headings" />
        </div>
      </div>
    </div>
  </div>
  <KunNull v-else description="未找到该文档" />
</template>
