<script setup lang="ts">
import type { Activity, WorkRef } from '#shared/utils/api/schemas'

const props = defineProps<{ activity: Activity; work: WorkRef }>()

const workId = computed(() => Number(props.work.id))
const stats = computed(() => props.activity.work_stats)
const targetUserId = computed(() => Number(props.activity.performer?.id ?? 0))

const { isLiked, isFavorited, ensureLoaded } = useMyGalgameInteractions()
onMounted(() => ensureLoaded([workId.value]))
</script>

<template>
  <ActivityCardShell
    :performer="activity.performer"
    :occurred-at="activity.occurred_at"
  >
    <div class="space-y-3">
      <p class="text-default-600 text-sm">
        创建了一个新的 Galgame，已经有
        {{ stats?.resource_count ?? 0 }} 个下载资源
      </p>

      <ActivityCardGalgameInfo :work="work" :digest="activity.work_digest" />

      <div class="flex items-center gap-2">
        <GalgameLike
          :work-id="workId"
          :like-count="stats?.like_count ?? 0"
          :is-liked="isLiked(workId)"
        />
        <GalgameFavorite
          :work-id="workId"
          :target-user-id="targetUserId"
          :favorite-count="stats?.favorite_count ?? 0"
          :is-favorited="isFavorited(workId)"
        />
        <KunLink
          underline="none"
          color="default"
          :to="`/galgame/${work.id}`"
          class-name="text-default-500 hover:text-primary ml-auto flex items-center gap-0.5 text-sm"
        >
          查看详情
          <KunIcon name="lucide:chevron-right" class="size-4" />
        </KunLink>
      </div>
    </div>
  </ActivityCardShell>
</template>
