<script setup lang="ts">
import type { WebsiteTag } from '#shared/utils/api/schemas'

definePageMeta({ key: (route) => route.path })

const route = useRoute()
const slug = computed(() => (route.params as { name: string }).name)

const canManageTaxonomy = useCan('website.edit')

const { data } = await useApi<WebsiteTag>(
  () => `website-tag:${slug.value}`,
  (api) =>
    api.GET('/website-tags/{website_tag_slug}', {
      params: { path: { website_tag_slug: slug.value } }
    })
)

const { data: websites } = await useWebsiteList(() =>
  data.value ? { website_tag_id: data.value.id } : {}
)
const sorted = computed(() =>
  data.value ? [...(websites.value ?? [])].sort(byScore) : []
)

if (data.value) {
  useKunSeoMeta({
    title: `${data.value.label}的 Galgame 网站`,
    description: data.value.description
  })
} else {
  useKunDisableSeo('未找到该网站标签')
}
</script>

<template>
  <div v-if="data" class="space-y-6">
    <KunHeader
      :name="`${data.label}的 Galgame 网站`"
      :description="data.description"
    >
      <template #endContent>
        <div class="space-y-3">
          <div class="flex items-center space-x-3">
            <KunChip color="primary">标签价值 {{ data.level }}</KunChip>
            <KunChip>{{ sorted.length }} 个网站</KunChip>
          </div>

          <div v-if="canManageTaxonomy" class="flex justify-end">
            <KunButton variant="light" href="/admin/website?tab=tag">
              管理标签
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

    <KunNull v-else :description="`${data.label} 标签下暂无网站`" />
  </div>
</template>
