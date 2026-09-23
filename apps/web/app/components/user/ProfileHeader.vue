<script setup lang="ts">
import { useIntersectionObserver } from '@vueuse/core'
import type { UserProfile } from '#shared/utils/api/schemas'
import { managementRoleLabel } from '~/constants/user'
import { toKunUser } from '~/utils/userRef'

const props = defineProps<{
  user: UserProfile
}>()

const currentUserId = usePersistUserStore().id
const isSelf = computed(() => currentUserId === Number(props.user.id))
const kunUser = computed(() => toKunUser(props.user))
const displayName = computed(() => props.user.name ?? '')

const metrics = computed(() => [
  { label: '萌萌点', value: props.user.moemoepoint, accent: true },
  { label: '话题', value: props.user.counts.topic_count },
  { label: 'Galgame', value: props.user.counts.published_galgame_count },
  { label: '评分', value: props.user.counts.galgame_rating_count },
  { label: '被赞', value: props.user.counts.received_like_count },
  { label: '被推', value: props.user.counts.received_upvote_count }
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
  <div ref="bannerRef">
    <KunCard :is-hoverable="false">
      <div class="flex items-start gap-4">
        <KunAvatar
          class-name="cursor-default shrink-0 relative"
          :is-navigation="false"
          size="original-sm"
          :user="kunUser"
          :disable-floating="true"
        />

        <div class="min-w-0 flex-1">
          <h1 class="flex flex-wrap items-center gap-2 text-2xl font-bold">
            <span class="truncate">{{ displayName }}</span>
            <KunButton
              v-if="!isSelf"
              variant="flat"
              size="xs"
              color="primary"
              class-name="gap-1"
              :href="`/message/user/${user.id}`"
            >
              <KunIcon name="lucide:message-circle" />
              私聊
            </KunButton>
            <ReportButton
              v-if="!isSelf"
              subject-kind="user"
              :subject-id="user.id"
              :snapshot="displayName"
              :subject-url="`${kungal.domain.main}/user/${user.id}`"
            />
          </h1>

          <div class="mt-2 flex flex-wrap items-center gap-2">
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
            class="text-default-600 mt-2 line-clamp-2 text-sm break-words"
          >
            {{ user.bio }}
          </p>

          <div class="text-default-500 mt-2 text-sm">
            注册于 <KunTime :time="user.created_at" type="date" show-year />
          </div>
        </div>
      </div>

      <div class="mt-5 flex flex-wrap gap-x-8 gap-y-3">
        <div v-for="m in metrics" :key="m.label" class="min-w-14">
          <div :class="cn('text-xl font-bold', m.accent && 'text-secondary')">
            {{ m.value }}
          </div>
          <div class="text-default-500 text-xs">{{ m.label }}</div>
        </div>
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
      <div v-show="collapsed" class="fixed inset-x-0 top-20 z-20">
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
                :disable-floating="true"
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
