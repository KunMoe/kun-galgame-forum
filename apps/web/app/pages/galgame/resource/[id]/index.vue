<script setup lang="ts">
import type { GalgameResource } from '#shared/utils/api/schemas'
import { problemMessage } from '#shared/utils/api/message'
import { settle, isMissingProblem } from '#shared/utils/api/problem'
import { contentPlainText } from '~/utils/contentPlainText'
import {
  resourceLanguageLabel,
  resourcePlatformLabel,
  resourceTypeLabel
} from '~~/shared/utils/galgameResourceVocab'

const route = useRoute()
const resourceId = computed(() => String((route.params as { id: string }).id))
const workName = useWorkName()
const api = useApiClient()

const { data, problem, pending, refresh } = await useKunApi<GalgameResource>(
  () => `galgame-resource:${resourceId.value}`,
  (api, { signal }) =>
    api.GET('/galgame-resources/{resource_id}', {
      params: { path: { resource_id: resourceId.value } },
      signal
    }),
  { critical: true }
)

const { gated, reveal } = useKunNsfwGate(() => data.value?.work?.is_nsfw)

const relocation = () =>
  settle(
    api.GET('/galgame-resource-relocations/{resource_id}', {
      params: { path: { resource_id: resourceId.value } }
    })
  )

if (!data.value && isMissingProblem(problem.value)) {
  const moved = await relocation()
  if (moved.ok) {
    await navigateTo(moved.data.url, { external: true, redirectCode: 301 })
  } else if (import.meta.server) {
    throw createError({
      statusCode: 404,
      statusMessage: '未找到对应的 Galgame 资源',
      fatal: true
    })
  }
}
watch(problem, async (failed) => {
  if (failed?.status !== 404) {
    return
  }
  const moved = await relocation()
  if (moved.ok) {
    await navigateTo(moved.data.url, { external: true })
  }
})

useKunSeoMeta(() => {
  const resource = data.value
  const work = resource?.work
  if (!resource || !work) {
    return problem.value || resource
      ? { noindex: true, title: '未找到 Galgame 资源' }
      : {}
  }

  const titleBase = workName(work)
  if (work.is_nsfw) {
    return { noindex: true, title: titleBase }
  }

  const typeLabel = resourceTypeLabel(resource.resource_type)
  const languageLabel = resource.resource_languages
    .map((lang) => resourceLanguageLabel(lang))
    .join(' ')
  const platformLabel = resource.resource_platforms
    .map((platform) => resourcePlatformLabel(platform))
    .join(' ')

  const description = `${typeLabel} · ${languageLabel} · ${platformLabel} · ${resource.size}`
  const note = contentPlainText(resource.content).trim()

  return {
    title: `${titleBase} ${typeLabel}资源下载`,
    description: note ? truncateRunes(note, 233) : description,
    ...(work.cover?.url ? { ogImage: work.cover.url } : {})
  }
})
</script>

<template>
  <div class="space-y-3">
    <KunNsfwGate v-if="data && gated" noun="资源" @reveal="reveal" />
    <template v-else-if="data">
      <GalgameResourceDetailHero v-if="data.work" :work="data.work" />

      <KunAdAIFYBanner class-name="hidden lg:block" />

      <div class="grid grid-cols-1 gap-3 lg:grid-cols-3">
        <GalgameResourceDetailPanel
          class="min-w-0 lg:col-span-2"
          :resource="data"
          :refresh="refresh"
        />

        <GalgameResourceDetailRecommendations
          v-if="data.work"
          class="min-w-0"
          :work-id="data.work.id"
          :current-id="data.id"
        />
      </div>

      <GalgameResourceCommentCommunityContainer
        :resource-id="Number(data.id)"
        :comment-count="data.comment_count"
      />
    </template>

    <GalgameResourceDetailSkeleton v-else-if="pending" />

    <KunNull
      v-else-if="problem && problem.status !== 404"
      :description="problemMessage(problem)"
    />

    <KunNull
      v-else
      description="未找到对应的 Galgame 资源，或该资源已被移除。"
    />
  </div>
</template>
