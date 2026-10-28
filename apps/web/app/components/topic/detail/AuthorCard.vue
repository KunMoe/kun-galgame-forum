<script setup lang="ts">
const props = defineProps<{
  user: {
    id: number
    name: string
    avatar: string
    moemoepoint: number
  }
}>()

const { id: currentUserId } = storeToRefs(usePersistUserStore())
const isSelf = computed(() => currentUserId.value === props.user.id)
</script>

<template>
  <div class="flex w-full flex-col items-center gap-3 px-1 pt-3">
    <KunAvatar
      :user="user"
      size="original"
      class-name="size-22 hover:scale-100"
      image-class-name="size-22"
    />

    <div class="flex min-w-0 flex-col items-center gap-1 pt-1">
      <KunLink
        underline="hover"
        color="default"
        :aria-label="user.name"
        :to="`/user/${user.id}`"
        class-name="max-w-full text-center text-lg font-semibold break-normal wrap-anywhere"
      >
        {{ user.name }}
      </KunLink>

      <p class="text-secondary flex items-center gap-1 text-sm">
        <KunIcon class="text-inherit" name="lucide:lollipop" />
        {{ user.moemoepoint }}
      </p>
    </div>

    <UserFollowButton v-if="!isSelf" :user-id="String(user.id)" block />
  </div>
</template>
