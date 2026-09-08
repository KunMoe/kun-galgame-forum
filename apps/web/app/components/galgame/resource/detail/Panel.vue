<script setup lang="ts">
import {
  KUN_GALGAME_RESOURCE_LANGUAGE_MAP,
  KUN_GALGAME_RESOURCE_PLATFORM_MAP,
  KUN_GALGAME_RESOURCE_TYPE_MAP,
  KUN_USER_TEXT_CHIP_CLASS
} from '~/constants/galgame'
import {
  GALGAME_RESOURCE_TYPE_ICON_MAP,
  GALGAME_RESOURCE_PLATFORM_ICON_MAP
} from '~/constants/galgameResource'

const props = defineProps<{
  galgame: GalgameResourceSummary
  resource: GalgameResource
  refresh: () => Promise<void>
}>()

const resourceTypeLabel = computed(
  () =>
    KUN_GALGAME_RESOURCE_TYPE_MAP[props.resource.type] || props.resource.type
)
const resourceLanguageLabel = computed(
  () =>
    KUN_GALGAME_RESOURCE_LANGUAGE_MAP[props.resource.language] ||
    props.resource.language
)
const resourcePlatformLabel = computed(
  () =>
    KUN_GALGAME_RESOURCE_PLATFORM_MAP[props.resource.platform] ||
    props.resource.platform
)
const galgameTitle = props.galgame.name
</script>

<template>
  <KunCard
    :is-hoverable="false"
    :is-transparent="false"
    content-class="space-y-4 h-full justify-start"
  >
    <KunHeader
      :name="`${galgameTitle} ${props.resource.platform === 'others' ? '' : resourcePlatformLabel} ${resourceLanguageLabel}${resourceTypeLabel}资源下载`"
      description="若资源链接失效, 或出现问题, 请点击反馈资源问题前往 Galgame 的评论区, 及时向资源发布者反馈或贡献新的下载资源。"
      scale="h1"
    />

    <div class="flex flex-wrap gap-2">
      <KunChip color="primary">
        <KunIcon :name="GALGAME_RESOURCE_TYPE_ICON_MAP[resource.type]" />
        {{ resourceTypeLabel }}
      </KunChip>
      <KunChip color="secondary">
        <KunIcon name="lucide:languages" />
        {{ resourceLanguageLabel }}
      </KunChip>
      <KunChip color="success">
        <KunIcon
          :name="GALGAME_RESOURCE_PLATFORM_ICON_MAP[resource.platform]"
        />
        {{ resourcePlatformLabel }}
      </KunChip>
      <KunChip color="warning" :class-name="KUN_USER_TEXT_CHIP_CLASS">
        <KunIcon name="lucide:database" />
        {{ resource.size }}
      </KunChip>
      <KunChip color="default">
        <KunIcon name="lucide:download" />
        {{ resource.download }}
      </KunChip>
      <KunChip color="default">
        <KunIcon name="lucide:eye" />
        {{ resource.view }}
      </KunChip>
    </div>

    <GalgameResourceDetailInfo
      :resource-type-label="resourceTypeLabel"
      :resource="resource"
      :refresh="refresh"
    />
  </KunCard>
</template>
