<script setup lang="ts">
import type { WebsiteCategory } from '#shared/utils/api/schemas'

definePageMeta({ key: (route) => route.path })

const route = useRoute()
const slug = computed(() => (route.params as { name: string }).name)

const canManageTaxonomy = useCan('website.edit')

const { data } = await useApi<WebsiteCategory>(
  () => `website-category:${slug.value}`,
  (api) =>
    api.GET('/website-categories/{website_category_slug}', {
      params: { path: { website_category_slug: slug.value } }
    })
)

const { data: websites } = await useWebsiteList(() =>
  data.value ? { website_category_id: data.value.id } : {}
)
const sorted = computed(() =>
  data.value ? [...(websites.value ?? [])].sort(byScore) : []
)

if (data.value) {
  useKunSeoMeta({
    title: data.value.label,
    description: data.value.description
  })
} else {
  useKunDisableSeo('未找到该网站分类')
}
</script>

<template>
  <div v-if="data" class="space-y-6">
    <KunHeader :name="data.label" :description="data.description">
      <template #endContent>
        <div class="space-y-3">
          <div class="flex items-center space-x-3">
            <KunChip color="primary">
              {{ `本资料库拥有 ${sorted.length} 个 ${data.label}` }}
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

    <div v-if="sorted.length">
      <div class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
        <WebsiteCard
          v-for="website in sorted"
          :key="website.id"
          :website="website"
        />
      </div>
    </div>

    <KunNull v-else :description="`${data.label} 分类下暂无网站`" />
  </div>
</template>
