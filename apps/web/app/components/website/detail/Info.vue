<script setup lang="ts">
import {
  KUN_WEBSITE_LANGUAGE_MAP,
  KUN_WEBSITE_NSFW_OPTIONS,
  KUN_WEBSITE_STATUS_OPTIONS,
  KUN_WEBSITE_STATUS_CHIP
} from '~/constants/galgameWebsite'
import type { Website } from '#shared/utils/api/schemas'

const props = defineProps<{
  data: Website
}>()

const utmLink = useUtmLink()

const statusLabel = computed(
  () =>
    KUN_WEBSITE_STATUS_OPTIONS.find(
      (option) => option.value === props.data.state
    )?.label ?? '正常'
)
const statusChip = computed(() => KUN_WEBSITE_STATUS_CHIP[props.data.state])
const nsfwLabel = computed(
  () => KUN_WEBSITE_NSFW_OPTIONS[props.data.is_nsfw ? 1 : 0]!.label
)
</script>

<template>
  <KunCard :is-transparent="false" :is-hoverable="false" class-name="p-6">
    <h3 class="text-default-900 mb-4 text-lg font-semibold">网站信息</h3>
    <div class="space-y-4">
      <div class="flex items-center justify-between">
        <span class="text-default-500 text-sm">分类</span>
        <KunLink
          :to="`/website-category/${data.website_category.slug}`"
          underline="none"
        >
          <KunChip class-name="cursor-pointer" color="primary">
            {{ data.website_category.label }}
          </KunChip>
        </KunLink>
      </div>

      <div class="flex items-center justify-between">
        <span class="text-default-500 text-sm">运行状态</span>
        <KunChip :color="statusChip ? statusChip.color : 'success'">
          {{ statusLabel }}
        </KunChip>
      </div>

      <div class="flex items-center justify-between">
        <span class="text-default-500 text-sm">语言</span>
        <KunChip color="secondary">
          {{ KUN_WEBSITE_LANGUAGE_MAP[data.language] ?? data.language }}
        </KunChip>
      </div>

      <div class="flex items-center justify-between">
        <span class="text-default-500 text-sm">年龄限制</span>
        <KunChip
          :variant="data.is_nsfw ? 'solid' : 'flat'"
          :color="data.is_nsfw ? 'danger' : 'success'"
        >
          {{ nsfwLabel }}
        </KunChip>
      </div>

      <div>
        <span class="text-default-500 text-sm">域名列表</span>
        <div
          v-for="(url, index) in data.urls"
          :key="index"
          class="mt-1 space-x-1"
        >
          <KunLink :to="utmLink(url)" class-name="font-mono">
            {{ url }}
          </KunLink>
          <KunButton
            :is-icon-only="true"
            variant="light"
            @click="useKunCopy(url)"
          >
            <KunIcon name="lucide:copy" />
          </KunButton>
        </div>
      </div>
    </div>
  </KunCard>
</template>
