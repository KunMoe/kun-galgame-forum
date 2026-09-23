<script setup lang="ts">
import {
  resourceLanguageLabel,
  resourcePlatformLabel,
  resourceTypeLabel
} from '~~/shared/utils/galgameResourceVocab'

const route = useRoute()
const resourceId = computed(() => Number((route.params as { id: string }).id))

const { data, refresh } = await useKunFetch<
  GalgameResourcePageData | 'not found'
>(`/galgame-resource/${resourceId.value}`, {
  query: { resource_id: resourceId }
})

if (data.value && data.value !== 'not found') {
  const titleBase = data.value.galgame.name

  if (data.value.galgame.content_limit === 'nsfw') {
    useKunDisableSeo(titleBase)
  } else {
    const resource = data.value.resource

    const typeLabel = resourceTypeLabel(resource.type)
    const languageLabel = resourceLanguageLabel(resource.language)
    const platformLabel = resourcePlatformLabel(resource.platform)

    const description = `${typeLabel} · ${languageLabel} · ${platformLabel} · ${resource.size}`

    useKunSeoMeta({
      title: `${titleBase} ${typeLabel}资源下载`,
      description: data.value.resource.note
        ? truncateRunes(markdownToText(data.value.resource.note).trim(), 233)
        : description,
      ogImage: getEffectiveBanner(data.value.galgame)
    })
  }
} else {
  useKunDisableSeo('未找到 Galgame 资源')
}
</script>

<template>
  <div v-if="data" class="space-y-3">
    <template v-if="data !== 'not found'">
      <GalgameResourceDetailHero :galgame="data.galgame" />

      <KunAdAIFYBanner class-name="hidden lg:block" />

      <div class="grid grid-cols-1 gap-3 lg:grid-cols-3">
        <GalgameResourceDetailPanel
          class="min-w-0 lg:col-span-2"
          :galgame="data.galgame"
          :resource="data.resource"
          :refresh="refresh"
        />

        <GalgameResourceDetailRecommendations
          class="min-w-0"
          :recommendations="data.recommendations"
        />
      </div>

      <GalgameResourceCommentCommunityContainer
        :resource-id="resourceId"
        :comment-count="data.resource.comment_count"
      />
    </template>

    <KunNull
      v-else
      description="未找到对应的 Galgame 资源，或该资源已被移除。"
    />
  </div>
</template>
