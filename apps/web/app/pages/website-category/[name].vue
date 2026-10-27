<script setup lang="ts">
import type { WebsiteCategory } from '#shared/utils/api/schemas'

definePageMeta({ key: (route) => route.path })

const route = useRoute()
const slug = computed(() => (route.params as { name: string }).name)

const canManageTaxonomy = useCan('website.edit')

const { data, problem, pending } = useKunApi<WebsiteCategory>(
  () => `website-category:${slug.value}`,
  (api) =>
    api.GET('/website-categories/{website_category_slug}', {
      params: { path: { website_category_slug: slug.value } }
    })
)

useKunSeoMeta(() => {
  if (data.value) {
    return { title: data.value.label, description: data.value.description }
  }
  return problem.value ? { noindex: true, title: '未找到该网站分类' } : {}
})
</script>

<template>
  <div>
    <WebsiteTaxonomyGrid
      v-if="data"
      :filter="{ website_category_id: data.id }"
      :empty-text="`${data.label} 分类下暂无网站`"
    >
      <template #header="{ count }">
        <KunHeader :name="data.label" :description="data.description">
          <template #endContent>
            <div class="space-y-3">
              <div class="flex items-center space-x-3">
                <KunChip color="primary">
                  {{ `本资料库拥有 ${count} 个 ${data.label}` }}
                </KunChip>
              </div>

              <div v-if="canManageTaxonomy" class="flex justify-end">
                <KunButton variant="light" href="/admin/website">
                  管理分类
                </KunButton>
              </div>
            </div>
          </template>
        </KunHeader>
      </template>
    </WebsiteTaxonomyGrid>

    <WebsiteTaxonomySkeleton v-else-if="pending" />

    <KunNull v-else description="未找到该网站分类" />
  </div>
</template>
