<script setup lang="ts">
import { managementRoleLabel } from '~/constants/user'
import { toKunUser } from '~/utils/userRef'

const props = defineProps<{
  userId: number | string
}>()

const emit = defineEmits<{
  navigate: []
}>()

const { entry, failed, load, adjustFollowers } = useUserCard(() => props.userId)
const { id: currentUserId } = storeToRefs(usePersistUserStore())

const profile = computed(() =>
  entry.value?.state === 'ready' ? entry.value.profile : null
)
const isSelf = computed(() => currentUserId.value === Number(props.userId))
const profileHref = computed(() => `/user/${props.userId}/info`)

const badges = computed(() => {
  const roles = profile.value?.roles ?? []
  return [
    roles.some((role) => role !== 'creator') && {
      label: managementRoleLabel(roles),
      color: 'primary' as const
    },
    roles.includes('creator') && { label: '创作者', color: 'warning' as const }
  ].filter((badge) => !!badge)
})

const stats = computed(() => {
  const counts = profile.value?.counts
  if (!counts) {
    return []
  }
  return [
    {
      label: '粉丝',
      value: counts.follower_count,
      href: `/user/${props.userId}/follow/followers`
    },
    {
      label: '关注',
      value: counts.following_count,
      href: `/user/${props.userId}/follow/following`
    },
    { label: '话题', value: counts.topic_count },
    { label: '回复', value: counts.reply_count },
    { label: '评论', value: counts.topic_comment_count },
    { label: '资源', value: counts.galgame_resource_count }
  ]
})

const onClick = (event: MouseEvent) => {
  if ((event.target as Element).closest('a[href]')) {
    emit('navigate')
  }
}

onMounted(() => {
  void load()
})
</script>

<template>
  <div class="w-80 max-w-[calc(100vw-2rem)] p-4" @click="onClick">
    <div v-if="profile" class="space-y-3">
      <div class="flex items-center gap-3">
        <KunAvatar :user="toKunUser(profile)" size="xl" class-name="shrink-0" />
        <div class="min-w-0 flex-1 space-y-1">
          <KunLink
            :to="profileHref"
            color="default"
            underline="hover"
            class-name="block truncate text-base font-semibold"
          >
            {{ toKunUser(profile).name }}
          </KunLink>
          <div class="flex flex-wrap items-center gap-1.5">
            <span
              class="text-secondary flex items-center gap-1 text-sm font-medium tabular-nums"
              title="萌萌点"
            >
              <KunIcon name="lucide:lollipop" />
              {{ profile.moemoepoint }}
            </span>
            <KunChip
              v-for="badge in badges"
              :key="badge.label"
              size="xs"
              variant="flat"
              :color="badge.color"
            >
              {{ badge.label }}
            </KunChip>
          </div>
        </div>
      </div>

      <p
        v-if="profile.bio"
        class="text-default-600 line-clamp-2 text-sm break-words"
      >
        {{ profile.bio }}
      </p>

      <dl class="bg-default-100 grid grid-cols-3 gap-y-2 rounded-lg py-2.5">
        <div
          v-for="stat in stats"
          :key="stat.label"
          class="flex flex-col-reverse items-center"
        >
          <dt class="text-default-500 text-xs">{{ stat.label }}</dt>
          <dd class="font-semibold tabular-nums">
            <KunLink
              v-if="stat.href"
              :to="stat.href"
              color="default"
              underline="none"
              class-name="hover:text-primary"
            >
              {{ stat.value ?? '—' }}
            </KunLink>
            <template v-else>{{ stat.value ?? '—' }}</template>
          </dd>
        </div>
      </dl>

      <div class="flex items-center gap-2">
        <template v-if="!isSelf">
          <UserFollowButton :user-id="profile.id" @change="adjustFollowers" />
          <KunButton
            variant="flat"
            size="xs"
            color="default"
            class-name="gap-1"
            :href="`/message/user/${profile.id}`"
          >
            <KunIcon name="lucide:message-circle" />
            私聊
          </KunButton>
        </template>
        <span class="text-default-500 ml-auto shrink-0 text-xs">
          <KunTime :time="profile.created_at" type="date" show-year /> 加入
        </span>
      </div>
    </div>

    <p
      v-else-if="entry?.state === 'missing'"
      class="text-default-500 flex items-center gap-2 text-sm"
    >
      <KunIcon name="lucide:user-x" />
      该用户不存在或已注销
    </p>

    <p
      v-else-if="failed"
      class="text-default-500 flex items-center gap-2 text-sm"
    >
      用户信息加载失败
      <KunButton size="xs" variant="light" @click="load">重试</KunButton>
    </p>

    <div v-else class="space-y-3" aria-busy="true">
      <div class="flex items-center gap-3">
        <KunSkeleton variant="circle" height="3.5rem" />
        <div class="flex-1 space-y-2">
          <KunSkeleton variant="text" width="60%" />
          <KunSkeleton variant="text" width="35%" />
        </div>
      </div>
      <KunSkeleton variant="rect" height="5.5rem" />
      <KunSkeleton variant="text" width="45%" />
    </div>
  </div>
</template>
