<script setup lang="ts">
import type { WebsiteTag } from '#shared/utils/api/schemas'

definePageMeta({ key: (route) => route.path })

const route = useRoute()
const slug = computed(() => (route.params as { name: string }).name)

const canManageTaxonomy = useCan('website.edit')

const { data, problem, pending } = useKunApi<WebsiteTag>(
  () => `website-tag:${slug.value}`,
  (api) =>
    api.GET('/website-tags/{website_tag_slug}', {
      params: { path: { website_tag_slug: slug.value } }
    }),
  { critical: true }
)

useKunSeoMeta(() => {
  if (data.value) {
    return {
      title: `${data.value.label}的 Galgame 网站`,
      description: data.value.description
    }
  }
  return problem.value ? { noindex: true, title: '未找到该网站标签' } : {}
})
</script>

<template>
  <div>
    <WebsiteTaxonomyGrid
      v-if="data"
      :filter="{ website_tag_id: data.id }"
      :empty-text="`${data.label} 标签下暂无网站`"
    >
      <template #header="{ count }">
        <KunHeader
          :name="`${data.label}的 Galgame 网站`"
          :description="data.description"
        >
          <template #endContent>
            <div class="space-y-3">
              <div class="flex items-center space-x-3">
                <KunChip color="primary">标签价值 {{ data.level }}</KunChip>
                <KunChip>{{ count }} 个网站</KunChip>
              </div>

              <div v-if="canManageTaxonomy" class="flex justify-end">
                <KunButton variant="light" href="/admin/website?tab=tag">
                  管理标签
                </KunButton>
              </div>
            </div>
          </template>
        </KunHeader>
      </template>
    </WebsiteTaxonomyGrid>

    <WebsiteTaxonomySkeleton v-else-if="pending" />

    <KunNull v-else description="未找到该网站标签" />
  </div>
</template>
