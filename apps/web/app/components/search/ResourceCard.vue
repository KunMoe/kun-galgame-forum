<script setup lang="ts">
import {
  GALGAME_RESOURCE_TYPE_ICON_MAP,
  GALGAME_RESOURCE_PLATFORM_ICON_MAP
} from '~/constants/galgameResource'
import { contentPlainText } from '~/utils/contentPlainText'
import {
  resourceLanguageLabel,
  resourcePlatformLabel,
  resourceRuntimeLabel,
  resourceTypeLabel
} from '~~/shared/utils/galgameResourceVocab'

const props = defineProps<{
  resource: SearchResultResource
  keywords?: string
}>()

const workName = useWorkName()
const title = computed(() =>
  props.resource.work ? workName(props.resource.work) : ''
)
const note = computed(() => contentPlainText(props.resource.content).trim())
const platform = computed(() => props.resource.resource_platforms[0])
const runtime = computed(() => props.resource.resource_runtimes[0])
const language = computed(() => props.resource.resource_languages[0])
</script>

<template>
  <KunLink
    color="default"
    underline="none"
    class-name="flex-col items-start w-full gap-1.5"
    :to="`/galgame/resource/${resource.id}`"
  >
    <div class="flex w-full items-baseline gap-2">
      <KunIcon
        :name="
          (platform && GALGAME_RESOURCE_PLATFORM_ICON_MAP[platform]) ||
          'lucide:ellipsis'
        "
        class="text-primary size-3.5 shrink-0 self-center"
      />
      <h3 class="hover:text-primary min-w-0 flex-1 truncate font-medium">
        <SearchHighlight :text="title" :keywords="keywords" />
      </h3>
      <span class="text-default-400 shrink-0 text-xs">
        <KunTime :time="resource.created_at" />
      </span>
    </div>

    <p v-if="note" class="text-default-600 line-clamp-2 w-full text-xs">
      <SearchHighlight :text="note" :keywords="keywords" />
    </p>

    <div
      class="text-default-500 flex w-full flex-wrap items-center gap-x-3 gap-y-1 text-xs"
    >
      <span class="flex items-center gap-1">
        <KunIcon
          :name="GALGAME_RESOURCE_TYPE_ICON_MAP[resource.resource_type]"
          class="size-3.5"
        />
        {{ resourceTypeLabel(resource.resource_type) }}
      </span>
      <span v-if="language">{{ resourceLanguageLabel(language) }}</span>
      <span v-if="platform">{{ resourcePlatformLabel(platform) }}</span>
      <span v-else-if="runtime">{{ resourceRuntimeLabel(runtime) }}</span>
      <span v-if="resource.size" class="flex items-center gap-1">
        <KunIcon name="lucide:database" class="size-3.5" />
        {{ resource.size }}
      </span>

      <span class="ml-auto flex shrink-0 items-center gap-3 tabular-nums">
        <span class="flex items-center gap-1">
          <KunIcon name="lucide:download" class="size-3.5" />
          {{ resource.download_count }}
        </span>
        <span class="flex items-center gap-1">
          <KunIcon name="lucide:eye" class="size-3.5" />
          {{ resource.view_count }}
        </span>
      </span>
    </div>
  </KunLink>
</template>
