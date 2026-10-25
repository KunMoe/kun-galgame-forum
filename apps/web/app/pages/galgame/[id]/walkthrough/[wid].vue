<script setup lang="ts">
import type { Walkthrough, Work } from '#shared/utils/api/schemas'

definePageMeta({ key: (route) => route.path })

const route = useRoute()
const { allowsNsfw, stanceKey } = useContentStance()

const params = route.params as { id: string; wid: string }
const workId = String(params.id)
const walkthroughId = String(params.wid)

const { data: walkthrough, problem } = await useApi<Walkthrough>(
  () => `walkthrough:${walkthroughId}`,
  (api, { signal }) =>
    api.GET('/walkthroughs/{walkthrough_id}', {
      params: { path: { walkthrough_id: walkthroughId } },
      signal
    })
)

if (problem.value?.status === 404 || (!walkthrough.value && !problem.value)) {
  throw createError({
    statusCode: 404,
    statusMessage: '未找到这份攻略',
    fatal: true
  })
}
if (walkthrough.value && walkthrough.value.work_id !== workId) {
  await navigateTo(
    `/galgame/${walkthrough.value.work_id}/walkthrough/${walkthroughId}`,
    {
      redirectCode: 301,
      replace: true
    }
  )
}

const { data: work } = await useApi<Work>(
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

const revealed = ref(false)
const isShowWalkthrough = computed(
  () => !walkthrough.value?.is_nsfw || allowsNsfw.value || revealed.value
)

if (walkthrough.value && work.value) {
  useGalgameWalkthroughSeo(walkthrough.value, work.value)
} else {
  useKunDisableSeo('攻略')
}
</script>

<template>
  <div>
    <template v-if="walkthrough && work">
      <div v-if="isShowWalkthrough" class="space-y-10">
        <GalgameWalkthroughHero :walkthrough="walkthrough" :work="work" />
        <GalgameWalkthroughContainer :walkthrough="walkthrough" :work="work" />
        <GalgameWalkthroughComment :walkthrough-id="Number(walkthrough.id)" />
      </div>
      <KunNsfwGate v-else noun="攻略" @reveal="revealed = true" />
    </template>
    <KunNull v-else description="攻略暂时无法加载, 请稍后再试" />
  </div>
</template>
