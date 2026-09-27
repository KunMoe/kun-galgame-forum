<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    user: KunUser & { moemoepoint: number }
    created: string | Date
    edited: string | Date | null
    topicId: number
    floor: number
    className?: string
    showAddition?: boolean
    showFollow?: boolean
  }>(),
  { className: '', showAddition: true, showFollow: false }
)

const { id: currentUserId } = storeToRefs(usePersistUserStore())
const canFollow = computed(
  () => props.showFollow && currentUserId.value !== props.user.id
)
</script>

<template>
  <div :class="cn('flex items-center gap-3', className)">
    <UserHoverCard :user-id="user.id">
      <KunAvatar size="lg" :user="user" />
    </UserHoverCard>

    <div class="w-full min-w-0">
      <div class="flex items-center justify-between gap-2">
        <div class="flex min-w-0 gap-2">
          <KunLink
            underline="hover"
            :to="`/user/${user.id}`"
            class-name="truncate"
          >
            {{ user.name }}
          </KunLink>
          <p class="text-secondary flex shrink-0 items-center gap-1">
            <KunIcon class="text-inherit" name="lucide:lollipop" />
            {{ user.moemoepoint }}
          </p>
        </div>

        <KunLink
          v-if="showAddition"
          color="default"
          underline="none"
          :to="`/topic/${topicId}?reply=${floor}`"
          class-name="text-default-400 font-bold"
        >
          #{{ floor }}
        </KunLink>

        <UserFollowButton v-if="canFollow" :user-id="String(user.id)" />
      </div>

      <div v-if="showAddition" class="text-xs text-gray-500 dark:text-gray-400">
        <span>
          发布于
          <KunTime :time="created" type="datetime" show-year />
        </span>
        <span v-if="edited" class="ml-2">
          (编辑于
          <KunTime :time="edited" type="datetime" show-year />)
        </span>
      </div>
    </div>
  </div>
</template>
