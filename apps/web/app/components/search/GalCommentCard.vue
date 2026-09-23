<script setup lang="ts">
import type { WallCommentSearchHit } from '#shared/utils/api/schemas'
import { toKunUser } from '~/utils/userRef'

const props = defineProps<{
  comment: WallCommentSearchHit
  keywords?: string
}>()

const WALL_LABEL: Record<WallCommentSearchHit['subject_type'], string> = {
  galgame: 'Galgame',
  galgame_rating: '游戏评分',
  galgame_resource: '下载资源',
  galgame_quiz: '游戏答题',
  toolset: 'Gal 工具',
  website: '网站'
}

const workName = useWorkName()

const label = computed(() => WALL_LABEL[props.comment.subject_type])
const title = computed(() =>
  props.comment.work ? workName(props.comment.work) : label.value
)
const author = computed(() => toKunUser(props.comment.author))
</script>

<template>
  <KunLink
    color="default"
    underline="none"
    :to="comment.subject_path"
    class-name="flex-col items-start w-full gap-1.5"
  >
    <div class="flex w-full items-baseline gap-2">
      <KunIcon
        class="text-primary size-3.5 shrink-0 self-center"
        name="lucide:message-circle"
      />
      <span class="min-w-0 flex-1 truncate text-sm font-medium">
        <SearchHighlight :text="title" :keywords="keywords" />
      </span>
      <KunChip size="sm" color="default" class-name="shrink-0">
        {{ label }}
      </KunChip>
    </div>

    <p
      class="border-primary text-default-700 line-clamp-2 w-full border-l-2 pl-2 text-sm"
    >
      <SearchHighlight :text="comment.excerpt" :keywords="keywords" />
    </p>

    <div class="text-default-500 flex w-full items-center gap-1.5 text-xs">
      <KunAvatar size="xs" :user="author" :is-navigation="false" />
      <span class="truncate">{{ author.name }}</span>
      <span class="ml-auto shrink-0">
        <KunTime :time="comment.created_at" />
      </span>
    </div>
  </KunLink>
</template>
