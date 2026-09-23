<script setup lang="ts">
import { KUN_USER_TEXT_CHIP_CLASS } from '~/constants/galgame'
import type { GalgameResource } from '#shared/utils/api/schemas'
import {
  resourceLanguageLabel as langLabel,
  resourcePlatformLabel as platLabel,
  resourceTypeLabel as typeLabel
} from '~~/shared/utils/galgameResourceVocab'
import {
  GALGAME_RESOURCE_TYPE_ICON_MAP,
  GALGAME_RESOURCE_PLATFORM_ICON_MAP
} from '~/constants/galgameResource'

const props = defineProps<{
  resource: GalgameResource
  refresh: () => Promise<void>
}>()

const downloadsHere = ref(0)

const workName = useWorkName()
const resourceTypeLabel = computed(() =>
  typeLabel(props.resource.resource_type)
)
const languageLabels = computed(() =>
  props.resource.resource_languages.map((lang) => langLabel(lang))
)
const platformLabels = computed(() =>
  props.resource.resource_platforms.map((platform) => platLabel(platform))
)
const galgameTitle = computed(() =>
  props.resource.work ? workName(props.resource.work) : ''
)
const headerName = computed(() => {
  const platform = platformLabels.value[0] ?? ''
  const language = languageLabels.value[0] ?? ''
  return `${galgameTitle.value} ${platform} ${language}${resourceTypeLabel.value}资源下载`
})
</script>

<template>
  <KunCard
    :is-hoverable="false"
    :is-transparent="false"
    content-class="space-y-4 h-full justify-start"
  >
    <KunHeader
      :name="headerName"
      description="若资源链接失效, 或出现问题, 请点击反馈资源问题前往 Galgame 的评论区, 及时向资源发布者反馈或贡献新的下载资源。"
      scale="h1"
    />

    <div class="flex flex-wrap gap-2">
      <KunChip color="primary">
        <KunIcon
          :name="GALGAME_RESOURCE_TYPE_ICON_MAP[resource.resource_type]"
        />
        {{ resourceTypeLabel }}
      </KunChip>
      <KunChip
        v-for="lang in resource.resource_languages"
        :key="lang"
        color="secondary"
      >
        <KunIcon name="lucide:languages" />
        {{ langLabel(lang) }}
      </KunChip>
      <KunChip
        v-for="platform in resource.resource_platforms"
        :key="platform"
        color="success"
      >
        <KunIcon :name="GALGAME_RESOURCE_PLATFORM_ICON_MAP[platform]" />
        {{ platLabel(platform) }}
      </KunChip>
      <KunChip color="warning" :class-name="KUN_USER_TEXT_CHIP_CLASS">
        <KunIcon name="lucide:database" />
        {{ resource.size }}
      </KunChip>
      <KunChip color="default">
        <KunIcon name="lucide:download" />
        {{ resource.download_count + downloadsHere }}
      </KunChip>
      <KunChip color="default">
        <KunIcon name="lucide:eye" />
        {{ resource.view_count }}
      </KunChip>
    </div>

    <GalgameResourceDetailInfo
      :resource-type-label="resourceTypeLabel"
      :resource="resource"
      :refresh="refresh"
      @downloaded="downloadsHere += 1"
    />
  </KunCard>
</template>
