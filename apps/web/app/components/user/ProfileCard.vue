<script setup lang="ts">
import { useIntersectionObserver } from '@vueuse/core'
import type { UserProfile } from '#shared/utils/api/schemas'
import { managementRoleLabel } from '~/constants/user'
import { toKunUser } from '~/utils/userRef'

const props = defineProps<{
  user: UserProfile
}>()

const { id: currentUserId } = storeToRefs(usePersistUserStore())
const isSelf = computed(
  () => !!currentUserId.value && currentUserId.value === Number(props.user.id)
)
const kunUser = computed(() => toKunUser(props.user))
const displayName = computed(() => props.user.name ?? '')

const followerCount = ref<number | null>(props.user.counts.follower_count)

watch(
  () => props.user.counts.follower_count,
  (value) => {
    followerCount.value = value
  }
)

const onFollowChange = (delta: number) => {
  if (typeof followerCount.value === 'number') {
    followerCount.value += delta
  }
}

const displayCount = (value: number | null) => (value === null ? '—' : value)

type Metric = {
  label: string
  value: number | null
  accent?: boolean
  href?: string
}

const playtime = useUserPlaytimeTotal(() => props.user.id)
const playtimeShown = computed(() =>
  playtime.value ? playtimeHeadline(playtime.value) : null
)

const metrics = computed<Metric[]>(() => [
  { label: '萌萌点', value: props.user.moemoepoint, accent: true },
  { label: '话题', value: props.user.counts.topic_count },
  { label: 'Galgame', value: props.user.counts.published_galgame_count },
  { label: '评分', value: props.user.counts.galgame_rating_count },
  { label: '被赞', value: props.user.counts.received_like_count },
  { label: '被推', value: props.user.counts.received_upvote_count },
  {
    label: '关注',
    value: props.user.counts.following_count,
    href: `/user/${props.user.id}/follow/following`
  },
  {
    label: '粉丝',
    value: followerCount.value,
    href: `/user/${props.user.id}/follow/followers`
  }
])

const bannerRef = ref<HTMLElement | null>(null)
const collapsed = ref(false)
useIntersectionObserver(
  bannerRef,
  ([entry]) => {
    collapsed.value = !entry?.isIntersecting
  },
  { rootMargin: '-80px 0px 0px 0px' }
)
</script>

<template>
  <div
    ref="bannerRef"
    class="lg:sticky lg:top-20 lg:max-h-[calc(100dvh-6rem)] lg:overflow-y-auto"
  >
    <KunCard :is-hoverable="false" content-class="gap-4">
      <div class="flex items-start gap-4 lg:flex-col lg:gap-3">
        <KunAvatar
          class-name="cursor-default shrink-0 relative"
          :is-navigation="false"
          size="original-sm"
          :user="kunUser"
        />

        <div class="min-w-0 flex-1 space-y-2">
          <h1 class="text-2xl font-bold break-words">{{ displayName }}</h1>

          <div class="flex flex-wrap items-center gap-2">
            <KunChip size="sm" color="primary">
              {{ managementRoleLabel(user.roles) }}
            </KunChip>
            <KunChip
              v-if="user.roles.includes('creator')"
              size="sm"
              color="warning"
            >
              创作者
            </KunChip>
          </div>

          <p
            v-if="user.bio"
            class="text-default-600 line-clamp-3 text-sm break-words"
          >
            {{ user.bio }}
          </p>

          <div class="text-default-500 text-sm">
            注册于 <KunTime :time="user.created_at" type="date" show-year />
          </div>
        </div>
      </div>

      <div v-if="!isSelf" class="flex flex-wrap items-center gap-2">
        <KunButton
          variant="flat"
          size="xs"
          color="primary"
          class-name="gap-1"
          :href="directMessagePath(user.id)"
        >
          <KunIcon name="lucide:message-circle" />
          私聊
        </KunButton>
        <UserFollowButton :user-id="user.id" @change="onFollowChange" />
        <ReportButton
          subject-kind="user"
          :subject-id="user.id"
          :snapshot="displayName"
          :subject-url="`${kungal.domain.main}/user/${user.id}`"
        />
      </div>

      <KunLink
        v-if="playtimeShown"
        :to="`/user/${user.id}/library`"
        color="default"
        underline="none"
        class-name="bg-default-100 hover:bg-default-200 flex items-center gap-3 rounded-lg px-3 py-2.5 transition-colors"
      >
        <KunIcon name="lucide:clock" class="text-primary size-6 shrink-0" />
        <span class="flex flex-col">
          <span class="text-xl font-bold tabular-nums">
            {{ playtimeShown.value }}
            <span class="text-default-500 text-sm font-normal">
              {{ playtimeShown.unit }}
            </span>
          </span>
          <span class="text-default-500 text-xs">总游玩时长</span>
        </span>
      </KunLink>

      <dl
        class="grid grid-cols-4 gap-x-4 gap-y-3 sm:grid-cols-8 lg:grid-cols-2"
      >
        <div v-for="m in metrics" :key="m.label" class="flex flex-col-reverse">
          <dt class="text-default-500 text-xs">{{ m.label }}</dt>
          <dd>
            <KunLink
              v-if="m.href"
              :to="m.href"
              color="default"
              underline="none"
              class-name="hover:text-primary text-xl font-bold"
            >
              {{ displayCount(m.value) }}
            </KunLink>
            <span
              v-else
              :class="cn('text-xl font-bold', m.accent && 'text-secondary')"
            >
              {{ displayCount(m.value) }}
            </span>
          </dd>
        </div>
      </dl>

      <div
        v-if="user.about_html"
        class="border-default-200 space-y-1 border-t pt-4"
      >
        <KunContent :content="user.about_html" compact />
        <KunLink
          v-if="isSelf"
          to="https://account.nextmoe.com/profile"
          target="_blank"
          rel="noopener"
          size="sm"
          color="default"
          underline="hover"
          class-name="text-default-500"
        >
          在账号中心编辑介绍
        </KunLink>
      </div>
    </KunCard>

    <Transition
      enter-active-class="transition duration-200 ease-out"
      enter-from-class="opacity-0 -translate-y-2"
      enter-to-class="opacity-100 translate-y-0"
      leave-active-class="transition duration-150 ease-in"
      leave-from-class="opacity-100 translate-y-0"
      leave-to-class="opacity-0 -translate-y-2"
    >
      <div v-show="collapsed" class="fixed inset-x-0 top-20 z-20 lg:hidden">
        <div class="flex justify-center">
          <div
            class="desktop-nav:mr-3 desktop-nav:ml-[104px] desktop-nav:px-0 w-full max-w-7xl min-w-0 px-2"
          >
            <div
              class="bg-content1 border-kun shadow-kun-sm flex items-center gap-3 rounded-xl border px-4 py-2 backdrop-blur-md"
            >
              <KunAvatar
                class-name="cursor-default shrink-0"
                :is-navigation="false"
                size="sm"
                :user="kunUser"
              />
              <span class="truncate font-semibold">{{ displayName }}</span>
              <KunChip
                size="sm"
                color="primary"
                class-name="hidden sm:inline-flex"
              >
                {{ managementRoleLabel(user.roles) }}
              </KunChip>

              <div
                class="text-default-600 ml-auto flex items-center gap-4 text-sm"
              >
                <span class="hidden sm:inline">
                  话题 {{ user.counts.topic_count }}
                </span>
                <span class="hidden sm:inline">
                  Galgame {{ user.counts.published_galgame_count }}
                </span>
                <span>
                  萌萌点 <b class="text-secondary">{{ user.moemoepoint }}</b>
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </div>
</template>
