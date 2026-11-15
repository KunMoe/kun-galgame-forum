<script setup lang="ts">
import { isMissingProblem } from '#shared/utils/api/problem'
import type { UserProfile } from '#shared/utils/api/schemas'
import {
  findUserNavSection,
  userNavTabs,
  userProfileNav,
  userSectionTabs
} from '~/constants/userProfileNav'

definePageMeta({
  key: (route) => (route.params as { id: string }).id,
  validate: (route) => /^\d+$/.test(String(route.params.id ?? ''))
})

const route = useRoute()

const userId = computed(() => (route.params as { id: string }).id)

const { data, problem, pending } = await useKunApi<UserProfile>(
  () => `user:${userId.value}`,
  (api, { signal }) =>
    api.GET('/users/{user_id}', {
      params: { path: { user_id: userId.value } },
      signal
    }),
  { critical: true }
)
useKunNotFound(() => isMissingProblem(problem.value), '未找到该用户')

const isSignInRequired = computed(
  () => problem.value?.code === 'PROFILE_SIGN_IN_REQUIRED'
)
const signIn = () => startOAuthLogin({ returnTo: route.fullPath })

const isNavLoading = ref(false)
const nuxtApp = useNuxtApp()
const MIN_VISIBLE_MS = 280
let navStartedAt = 0
let clearTimer: ReturnType<typeof setTimeout> | null = null
// Only a move between sections dims the panel. Paging a list changes nothing
// but the query, and the inert this sets dropped the page button the reader
// had just pressed to <body>.
const stopNavHooks = [
  useRouter().beforeEach((to, from) => {
    if (to.path === from.path) {
      return
    }
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

const nav = computed(() =>
  data.value ? userProfileNav(data.value, isOwner.value) : []
)
const active = computed(() => findUserNavSection(nav.value, route.path))
const hasSectionTabs = computed(
  () => (active.value?.item.sections.length ?? 0) > 1
)

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
    <div
      v-if="data"
      class="grid grid-cols-1 items-start gap-4 lg:grid-cols-[18rem_minmax(0,1fr)] lg:gap-6"
    >
      <UserProfileCard :user="data" />

      <div class="min-w-0 space-y-4 lg:min-h-[calc(100dvh-6rem)]">
        <KunTab
          :items="userNavTabs(nav)"
          :model-value="active?.item.value ?? ''"
          variant="underlined"
          color="primary"
          scrollable
        />

        <div v-if="hasSectionTabs && active">
          <KunTab
            :items="userSectionTabs(active.item)"
            :model-value="active.section.value"
            variant="pills"
            color="primary"
            size="sm"
            scrollable
          />
        </div>

        <KunLoadingDim :loading="isNavLoading" :delay="0">
          <NuxtPage :user="data" />
        </KunLoadingDim>
      </div>
    </div>

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
