<script setup lang="ts">
import type { Activity, CollectionSummary } from '#shared/utils/api/schemas'

const props = defineProps<{
  activity: Activity
  collection: CollectionSummary
}>()

const nameOf = useWorkName()

const title = computed(() =>
  collectionDisplayName(props.collection, props.collection.owner.name)
)
const blurb = computed(() => collectionBlurb(props.collection, nameOf))
</script>

<template>
  <ActivityCardShell
    :performer="activity.performer"
    :occurred-at="activity.occurred_at"
  >
    <div
      v-if="activity.activity_type === 'collection_creation'"
      class="space-y-2"
    >
      <p class="text-default-600 flex items-center gap-1.5">
        <KunIcon name="lucide:folder-plus" class="text-primary size-4" />
        创建了一个新的收藏夹
      </p>
      <div class="bg-default-100 rounded-lg">
        <GalgameCollectionRowCard
          :collection="collection"
          :show-owner="false"
        />
      </div>
    </div>

    <div
      v-else-if="
        activity.activity_type === 'collection_work_addition' && activity.work
      "
      class="flex gap-3"
    >
      <ActivityCardWorkCover :work="activity.work" />
      <div class="min-w-0 flex-1 space-y-1.5">
        <p class="text-default-700">
          收藏了
          <KunLink
            underline="hover"
            :to="`/galgame/${activity.work.id}`"
            class-name="font-medium"
          >
            {{ nameOf(activity.work) }}
          </KunLink>
        </p>
        <KunLink
          underline="none"
          color="default"
          :to="activity.path"
          class-name="hover:text-primary inline-flex items-center gap-1.5 text-base font-semibold"
        >
          <KunIcon
            name="lucide:folder-heart"
            class="text-primary size-4 shrink-0"
          />
          {{ title }}
        </KunLink>
        <p
          v-if="blurb"
          class="text-default-500 line-clamp-2 text-sm break-words"
        >
          {{ blurb }}
        </p>
        <GalgameCollectionStats :collection="collection" />
      </div>
    </div>

    <div v-else class="space-y-2">
      <ActivityCollapse :max-height="300">
        <p class="text-default-700 text-base break-all whitespace-pre-line">
          {{ activity.excerpt_markdown }}
        </p>
      </ActivityCollapse>
      <NuxtLink
        :to="activity.path"
        class="bg-default-100 hover:bg-default-200 block space-y-1 rounded-lg px-3 py-2 transition-colors"
      >
        <span class="flex items-center gap-1.5 font-semibold">
          <KunIcon
            name="lucide:folder-heart"
            class="text-primary size-4 shrink-0"
          />
          {{ title }}
        </span>
        <span v-if="blurb" class="text-default-500 line-clamp-1 block text-sm">
          {{ blurb }}
        </span>
      </NuxtLink>
    </div>
  </ActivityCardShell>
</template>
