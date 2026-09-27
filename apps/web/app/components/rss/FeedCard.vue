<script setup lang="ts">
import {
  kunFeedUrl,
  type KunFeedFormat,
  type KunFeedUrlParams
} from '#shared/utils/feedUrl'

const props = defineProps<{
  title: string
  description: string
  path: string
  params?: KunFeedUrlParams
}>()

const config = useRuntimeConfig()
const baseUrl = computed(() => config.public.KUN_GALGAME_URL || '')
const rssUrl = computed(() =>
  kunFeedUrl(baseUrl.value, props.path, 'xml', props.params ?? {})
)
const formatUrl = (format: KunFeedFormat) =>
  kunFeedUrl(baseUrl.value, props.path, format, props.params ?? {})
</script>

<template>
  <KunCard :is-hoverable="false">
    <div class="space-y-3">
      <h2 class="text-xl">{{ title }}</h2>
      <p class="text-default-500">{{ description }}</p>
      <slot />
      <div class="flex flex-wrap items-center gap-2">
        <KunLink :href="rssUrl" target="_blank">{{ rssUrl }}</KunLink>
        <KunCopy :text="rssUrl" name="复制 RSS 链接" />
      </div>
      <div class="text-default-500 flex flex-wrap items-center gap-3">
        <span>其它格式:</span>
        <KunLink :href="formatUrl('atom')" target="_blank">Atom</KunLink>
        <KunLink :href="formatUrl('json')" target="_blank">JSON Feed</KunLink>
      </div>
    </div>
  </KunCard>
</template>
