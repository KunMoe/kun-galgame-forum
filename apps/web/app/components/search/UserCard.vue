<script setup lang="ts">
import type { UserSearchHit } from '#shared/utils/api/schemas'
import { KUN_USER_ROLE_MAP } from '~/constants/user'
import { toKunUser } from '~/utils/userRef'

const props = defineProps<{
  user: UserSearchHit
  keywords?: string
}>()

const avatarUser = computed(() => toKunUser(props.user))

// One chip, the highest rank the account holds — a name followed by four of
// them is a badge collection, not an identity.
const ROLE_RANK = ['ren', 'admin', 'moderator', 'creator']

const role = computed(() =>
  ROLE_RANK.find((name) =>
    props.user.roles.includes(name as UserSearchHit['roles'][number])
  )
)
</script>

<template>
  <KunCard :href="`/user/${user.id}`" :is-hoverable="true" padding="none">
    <div class="flex w-full gap-3 p-3">
      <KunAvatar
        :disable-floating="true"
        :user="avatarUser"
        :is-navigation="false"
        size="lg"
        class-name="shrink-0"
      />

      <div class="min-w-0 flex-1 space-y-1">
        <div class="flex items-center gap-2">
          <span class="truncate text-sm font-medium">
            <SearchHighlight :text="avatarUser.name" :keywords="keywords" />
          </span>
          <KunChip v-if="role" size="xs" color="primary">
            {{ KUN_USER_ROLE_MAP[role] }}
          </KunChip>
        </div>

        <p
          v-if="user.bio"
          class="text-default-600 line-clamp-2 text-xs break-all"
        >
          <SearchHighlight :text="user.bio" :keywords="keywords" />
        </p>

        <div
          class="text-default-500 flex flex-wrap items-center gap-x-3 text-xs tabular-nums"
        >
          <span class="flex items-center gap-1">
            <KunIcon name="lucide:square-gantt-chart" class="size-3.5" />
            {{ user.topic_count }}
          </span>
          <span class="flex items-center gap-1">
            <KunIcon name="carbon:reply" class="size-3.5" />
            {{ user.reply_count }}
          </span>
          <span v-if="user.registered_at" class="flex items-center gap-1">
            <KunIcon name="lucide:calendar" class="size-3.5" />
            <KunTime :time="user.registered_at" type="date" show-year />
          </span>
        </div>
      </div>
    </div>
  </KunCard>
</template>
