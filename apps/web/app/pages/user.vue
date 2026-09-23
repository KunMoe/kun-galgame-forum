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

const { data } = await useApi<UserProfile>(
  () => `user:${userId.value}`,
  (api, { signal }) =>
    api.GET('/users/{user_id}', {
      params: { path: { user_id: userId.value } },
      signal
    })
)

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

if (data.value) {
  useKunSeoMeta({
    title: data.value.name ?? '',
    description: data.value.bio ?? ''
  })
} else {
  useKunDisableSeo('未找到该用户')
}
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

    <KunNull v-else description="未找到该用户" />
  </div>
</template>
