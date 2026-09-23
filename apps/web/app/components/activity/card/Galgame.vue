<script setup lang="ts">
const props = defineProps<{ activity: ActivityItem }>()

const data = computed(
  () => props.activity.data as GalgameActivityData | undefined
)
const workId = computed(() => data.value?.galgame_id ?? 0)
const detailLink = computed(() =>
  workId.value ? `/galgame/${workId.value}` : props.activity.link
)

const { isLiked, isFavorited, ensureLoaded } = useMyGalgameInteractions()
onMounted(() => ensureLoaded(workId.value ? [workId.value] : []))
</script>

<template>
  <ActivityCardShell :actor="activity.actor" :timestamp="activity.timestamp">
    <div class="space-y-3">
      <p class="text-default-600 text-sm">
        创建了一个新的 Galgame,已经有 {{ data?.resource_count ?? 0 }} 个下载资源
      </p>

      <ActivityCardGalgameInfo :activity="activity" />

      <div class="flex items-center gap-2">
        <GalgameLike
          :work-id="workId"
          :target-user-id="activity.actor?.id ?? 0"
          :like-count="data?.like_count ?? 0"
          :is-liked="isLiked(workId)"
        />
        <GalgameFavorite
          :work-id="workId"
          :target-user-id="activity.actor?.id ?? 0"
          :favorite-count="data?.favorite_count ?? 0"
          :is-favorited="isFavorited(workId)"
        />
        <KunLink
          underline="none"
          color="default"
          :to="detailLink"
          class-name="text-default-500 hover:text-primary ml-auto flex items-center gap-0.5 text-sm"
        >
          查看详情
          <KunIcon name="lucide:chevron-right" class="size-4" />
        </KunLink>
      </div>
    </div>
  </ActivityCardShell>
</template>
