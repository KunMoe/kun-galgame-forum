<script setup lang="ts">
import type { Walkthrough, Work } from '#shared/utils/api/schemas'

definePageMeta({ key: (route) => route.path })

const route = useRoute()
const { allowsNsfw, stanceKey } = useContentStance()

const params = route.params as { id: string; wid: string }
const workId = String(params.id)
const walkthroughId = String(params.wid)

const { data: work, pending: workPending } = useKunApi<Work>(
  () => `work:${workId}:${stanceKey.value}`,
  (api, { signal }) =>
    api.GET('/works/{work_id}', {
      params: {
        path: { work_id: workId },
        query: { include_nsfw: allowsNsfw.value }
      },
      signal
    })
)

const {
  data: walkthrough,
  problem,
  pending
} = await useKunApi<Walkthrough>(
  () => `walkthrough:${walkthroughId}`,
  (api, { signal }) =>
    api.GET('/walkthroughs/{walkthrough_id}', {
      params: { path: { walkthrough_id: walkthroughId } },
      signal
    })
)

useKunNotFound(() => problem.value?.status === 404, '未找到这份攻略')

const redirectToOwner = (owner: string) =>
  navigateTo(`/galgame/${owner}/walkthrough/${walkthroughId}`, {
    redirectCode: 301,
    replace: true
  })
if (walkthrough.value && walkthrough.value.work_id !== workId) {
  await redirectToOwner(walkthrough.value.work_id)
}
watch(walkthrough, (next) => {
  if (next && next.work_id !== workId) {
    redirectToOwner(next.work_id)
  }
})

const revealed = ref(false)
const isRouteHintsDismissed = ref(false)
const isShowWalkthrough = computed(
  () => !walkthrough.value?.is_nsfw || allowsNsfw.value || revealed.value
)

useGalgameWalkthroughSeo(
  () => walkthrough.value,
  () => work.value
)
</script>

<template>
  <div>
    <template v-if="walkthrough && work">
      <div v-if="isShowWalkthrough" class="space-y-10">
        <GalgameWalkthroughHero :walkthrough="walkthrough" :work="work" />
        <GalgameWalkthroughContainer
          v-if="!walkthrough.has_in_game_route_hints || isRouteHintsDismissed"
          :walkthrough="walkthrough"
          :work="work"
        />
        <GalgameWalkthroughRouteHintNotice
          v-else
          @reveal="isRouteHintsDismissed = true"
        />
        <GalgameWalkthroughComment :walkthrough-id="Number(walkthrough.id)" />
      </div>
      <KunNsfwGate v-else noun="攻略" @reveal="revealed = true" />
    </template>
    <GalgameWalkthroughDetailSkeleton v-else-if="pending || workPending" />
    <KunNull v-else description="攻略暂时无法加载, 请稍后再试" />
  </div>
</template>
