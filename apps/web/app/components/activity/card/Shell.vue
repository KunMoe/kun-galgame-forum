<script setup lang="ts">
import type { UserRef } from '#shared/utils/api/schemas'
import { toKunUser } from '~/utils/userRef'

const props = defineProps<{
  performer: UserRef | null
  occurredAt: string
}>()

const user = computed(() =>
  props.performer ? toKunUser(props.performer) : null
)
</script>

<template>
  <div class="flex w-full gap-3">
    <KunAvatar v-if="user" :user="user" />

    <div class="min-w-0 flex-1 space-y-2">
      <div class="flex flex-wrap items-center gap-x-2 gap-y-1 text-sm">
        <KunLink
          v-if="user"
          :to="`/user/${user.id}`"
          underline="none"
          color="default"
          class-name="text-default-800 hover:text-primary font-medium"
        >
          {{ user.name }}
        </KunLink>
        <span class="text-default-500">
          <KunTime :time="occurredAt" />
        </span>
        <slot name="meta" />
      </div>

      <slot />
    </div>
  </div>
</template>
