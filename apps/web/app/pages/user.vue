<script setup lang="ts">
import type { UserProfile } from '#shared/utils/api/schemas'
import {
  kunUserMainNav,
  userSegmentGroup,
  userSegmentHref,
  userTopicGroupOptions,
  userGalgameGroupOptions
} from '~/constants/user'

definePageMeta({ key: (route) => (route.params as { id: string }).id })

const route = useRoute()

const userId = computed(() => (route.params as { id: string }).id)

const { data, problem, pending } = useKunApi<UserProfile>(
  () => `user:${userId.value}`,
  (api, { signal }) =>
    api.GET('/users/{user_id}', {
      params: { path: { user_id: userId.value } },
      signal
    }),
  { critical: true }
)

const isSignInRequired = computed(
  () => problem.value?.code === 'PROFILE_SIGN_IN_REQUIRED'
)
const signIn = () => startOAuthLogin({ returnTo: route.fullPath })

const isNavLoading = ref(false)
const nuxtApp = useNuxtApp()
const MIN_VISIBLE_MS = 280
let navStartedAt = 0
let clearTimer: ReturnType<typeof setTimeout> | null = null
const stopNavHooks = [
  nuxtApp.hook('page:loading:start', () => {
    if (clearTimer) {
      clearTimeout(clearTimer)
      clearTimer = null
    }
    navStartedAt = Date.now()
    isNavLoading.value = true
  }),
  nuxtApp.hook('page:loading:end', () => {
    const wait = Math.max(0, MIN_VISIBLE_MS - (Date.now() - navStartedAt))
    clearTimer = setTimeout(() => {
      isNavLoading.value = false
    }, wait)
  })
]
onScopeDispose(() => {
  stopNavHooks.forEach((stop) => stop())
  if (clearTimer) {
    clearTimeout(clearTimer)
  }
})

const { id: storeUid } = storeToRefs(usePersistUserStore())
const isOwner = computed(
  () => !!storeUid.value && Number(userId.value) === storeUid.value
)

const activeSegment = computed(() => {
  const m = route.path.match(/^\/user\/\d+\/([^/]+)/)
  return m ? m[1]! : 'activity'
})
const activeGroup = computed(() => userSegmentGroup(activeSegment.value))
const goToSegment = (seg: string) =>
  navigateTo(userSegmentHref(Number(userId.value), seg))

useKunSeoMeta(() => {
  if (data.value) {
    return { title: data.value.name ?? '', description: data.value.bio ?? '' }
  }
  if (isSignInRequired.value) {
    return { noindex: true, title: '登录后查看该用户' }
  }
  return problem.value ? { noindex: true, title: '未找到该用户' } : {}
})
</script>

<template>
  <div class="space-y-4">
    <template v-if="data">
      <UserProfileHeader :user="data" />

      <div
        class="grid grid-cols-1 items-start gap-4 sm:grid-cols-[auto_minmax(0,1fr)]"
      >
        <div class="sm:hidden">
          <KunTab
            :items="kunUserMainNav(Number(data.id), isOwner)"
            :model-value="activeGroup"
            variant="solid"
            color="primary"
            size="sm"
            scrollable
          />
        </div>

        <div class="hidden self-start sm:sticky sm:top-36 sm:block">
          <KunTab
            :items="kunUserMainNav(Number(data.id), isOwner)"
            :model-value="activeGroup"
            orientation="vertical"
            variant="underlined"
            color="primary"
          />
        </div>

        <div class="min-w-0 sm:min-h-[calc(100dvh-9rem)]">
          <div
            v-if="activeGroup === 'topic' || activeGroup === 'galgame'"
            class="mb-4"
          >
            <KunRadioGroup
              :model-value="activeSegment"
              :options="
                activeGroup === 'topic'
                  ? userTopicGroupOptions
                  : userGalgameGroupOptions(isOwner)
              "
              variant="pill"
              orientation="horizontal"
              color="primary"
              size="sm"
              @change="goToSegment"
            />
          </div>

          <KunLoadingDim :loading="isNavLoading" :delay="0">
            <NuxtPage :user="data" />
          </KunLoadingDim>
        </div>
      </div>
    </template>

    <UserProfileSkeleton v-else-if="pending" />

    <div
      v-else-if="isSignInRequired"
      class="flex flex-col items-center gap-3 py-12"
    >
      <KunNull description="该用户设置了仅登录用户可查看主页" />
      <KunButton color="primary" @click="signIn">登录</KunButton>
    </div>

    <KunNull v-else description="未找到该用户" />
  </div>
</template>
