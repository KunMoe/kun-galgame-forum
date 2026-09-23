<script setup lang="ts">
import type { operations } from '#shared/types/api/v1'
import { KUN_TOPIC_SECTION } from '~/constants/topic'
import { KUN_TOPIC_SECTION_DESCRIPTION_MAP } from '~/constants/section'

const route = useRoute()
type SectionSlug = NonNullable<
  NonNullable<operations['listTopics']['parameters']['query']>['section']
>

const section = computed(() => (route.params as { section: string }).section)
const knownSection = computed(() =>
  Object.hasOwn(KUN_TOPIC_SECTION, section.value)
    ? (section.value as SectionSlug)
    : null
)

const KUN_TOPIC_CATEGORY: Record<string, string> = {
  g: 'Galgame',
  t: '技术交流',
  o: '其它话题'
}

useKunSeoMeta({
  title: `${KUN_TOPIC_CATEGORY[section.value.slice(0, 1)]} - ${KUN_TOPIC_SECTION[section.value]}`,
  description:
    KUN_TOPIC_SECTION_DESCRIPTION_MAP[section.value.toLocaleLowerCase()]
})
</script>

<template>
  <SectionContainer v-if="knownSection" :section="knownSection" />
  <KunNull v-else description="没有这个版块" />
</template>
