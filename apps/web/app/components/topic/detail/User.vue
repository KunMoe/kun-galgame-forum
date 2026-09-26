<script setup lang="ts">
withDefaults(
  defineProps<{
    user: KunUser & { moemoepoint: number }
    created: string | Date
    edited: string | Date | null
    topicId: number
    floor: number
    className?: string
    showAddition?: boolean
  }>(),
  { className: '', showAddition: true }
)
</script>

<template>
  <div :class="cn('flex items-center gap-2', className)">
    <UserHoverCard :user-id="user.id">
      <KunAvatar size="lg" :user="user" />
    </UserHoverCard>

    <div class="w-full">
      <div class="flex items-center justify-between">
        <div class="flex gap-2">
          <KunLink underline="hover" :to="`/user/${user.id}`">
            {{ user.name }}
          </KunLink>
          <p class="text-secondary flex items-center gap-1">
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
