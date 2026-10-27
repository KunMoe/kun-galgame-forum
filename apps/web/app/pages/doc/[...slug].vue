<script setup lang="ts">
import { contentHeadings } from '~/utils/contentPlainText'

const route = useRoute()

const docSlug = computed(() => {
  const slug = route.params.slug
  return Array.isArray(slug) ? slug.join('/') : slug || ''
})

const { data, problem, pending } = useKunApi(
  () => `doc:${docSlug.value}`,
  (api, { signal }) =>
    api.GET('/docs/{doc_slug}', {
      params: { path: { doc_slug: docSlug.value } },
      signal
    })
)

const headings = computed(() => contentHeadings(data.value?.content))

useKunSeoMeta(() => {
  const doc = data.value
  if (!doc) {
    return problem.value ? { noindex: true, title: '未找到该文档' } : {}
  }
  return {
    title: doc.title,
    description: doc.description,
    ogImage: doc.banner?.url,
    ogType: 'article',
    articleAuthor: [`${kungal.domain.main}/user/${doc.author.id}`],
    articlePublishedTime: doc.published_at,
    articleModifiedTime: doc.edited_at ?? undefined
  }
})
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
  <div
    v-else-if="pending"
    class="min-h-[calc(100dvh-6rem)] pb-6"
    aria-hidden="true"
  >
    <div class="space-y-6 pl-0 lg:pr-67 xl:pl-67">
      <div class="space-y-3">
        <KunSkeleton variant="text" width="70%" class-name="text-3xl" />
        <KunSkeleton variant="text" width="14rem" class-name="text-sm" />
        <KunSkeleton height="16rem" rounded="lg" />
      </div>
      <div>
        <KunSkeleton
          v-for="n in 10"
          :key="n"
          variant="text"
          :width="n === 10 ? '55%' : '100%'"
          class-name="leading-7"
        />
      </div>
    </div>
  </div>
  <KunNull v-else description="未找到该文档" />
</template>
